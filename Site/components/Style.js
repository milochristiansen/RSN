
import { h, css } from "/header.js"

// This is a very scuffed version of Styled Components.
const Style = new Proxy(
	// It would be pretty easy to add another layer of function and bind the create element function. That way
	// this could be used with other frameworks.
	(typ) => {
		// This will become the render function, after a little help from bind.
		let render = (dprops, genclass, props) => {
			// Combine prop sets
			let combined = {...dprops, ...props}

			// Strip props we otherwise handle
			let {as, children, ["class"]:no, ...remaining} = combined

			// Handle the special "as" prop.
			if (typeof combined.as == "string") {
				typ = combined.as
			}

			return h(typ, {class: [genclass, combined.class].join(" "), ...remaining}, ...combined.children)
		}

		// This parses the CSS and then binds it into the render function along with the default props.
		let generate = (dprops, strings, ...exprs) => {
			const generated = css(strings, ...exprs)

			return render.bind(null, dprops, generated)
		}

		// And here we have the interface. Two ways to make this work, call it directly or use a "props"
		// property to bind a custom default props object.
		return new Proxy(generate, {
			get(self, prop, proxy) {
				if (prop == "props") {
					return (props) => {
						return self.bind(null, props)
					}
				}
				return undefined
			},
			apply(self, tthis, args) {
				// We don't have to bind anything here, just do the call.
				return self({}, ...args)
			}
		})
	}, {
		// This exists to allow the cleaner syntax where the element type is a prop instead of an argument.
		get(self, prop, proxy) {
			return self(prop)
		}
	}
)

/*
import Style from "/components/Style.js"

// You can do things the basic way
let Test = Style.p`
	color: red;
`

// Or you can bind in some props with default values. Props specified at the usage site override these.
let Link = Style.a.props({href:"https://example.com"})`
	color: red;
`

<${Test}>Some red text.<//>

// If for some reason you need to override the element type at the call site, you can use the special "as" prop.
<${Test} as="div">Some red text.<//>
*/

export default Style
