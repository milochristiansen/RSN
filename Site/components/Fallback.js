import { html } from "/header.js"
import { Style } from "/components/Style.js"

const FallbackContainer = Style.div`
	width: 100%;
	font-size: 32px;
	text-align: center;

	color: var(--heading-color);
`

const Fallback = (props) => {
	return html`
		<${FallbackContainer}>${props.children}<//>
	`
}

export { Fallback }
export default Fallback
