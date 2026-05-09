
import { html, css } from "/header.js"

const style = css`
	width: 100%;
	font-size: 32px;
	text-align: center;

	color: var(--heading-color);
`

const Fallback = (props) => {
	return html`
		<div class=${style}>${props.children}</div>
	`
}

export { Fallback }
export default Fallback
