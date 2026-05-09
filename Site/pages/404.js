import { html, Meta, Title, Style } from "/header.js"

const Centered = Style.p`
	width: 100%;
	text-align: center;
`

export const E404 = (props) => {
	return html`
		<${Title} text="RSN - 404" />
		<${Meta} k="description" v="404 - Page not found." />

		<${Centered}>The page you were looking for is not on this server.<//>
		<${Centered}>You may <a href="/">return to the main page</a> or select a destination from the links in the header.<//>
	`
}

export default E404
