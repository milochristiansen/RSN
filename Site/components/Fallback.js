
import { html, css } from "/header.js"

const style = css`
	width: 100%;
	font-size: 32px;
	text-align: center;
`

function Fallback(props) {
	return html`
		<span class=${style}>${props.children}</span>
	`
}

export default Fallback
