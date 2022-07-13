
import { html, createContext, Component } from "/header.js"

const ThemeContext = createContext({refresh: () => {}, theme: "dark"})

// Alias to make the API orthogonal
const ThemeConsumer = ThemeContext.Consumer

class ThemeProvider extends Component {
	constructor() {
		super();
	
		const theme = localStorage.getItem('theme') == "light" ? "light" : "dark"
		this.state = {data: {toggle: this.toggle, theme: theme}}
		document.documentElement.setAttribute("data-theme", theme);
	}

	toggle = (evnt) => {
		this.setState(function(state, props) {
			const theme = state.data.theme == "light" ? "dark" : "light"
			document.documentElement.setAttribute("data-theme", theme);
			return {
				data: {toggle: this.toggle, data: theme}
			};
		})
		evnt.preventDefault();
	}

	render(props, state) {
		return html`
			<${ThemeContext.Provider} value=${state.data}>
				${props.children}
			<//>
		`
	}
}

export { ThemeContext, ThemeConsumer, ThemeProvider }
