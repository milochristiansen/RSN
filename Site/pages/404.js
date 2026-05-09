import { html, Meta, Title } from "/header.js"

let css = `
	width: 100%;
	text-align: center;
`

export const E404 = (props) => {
	return html`
		<${Title} text="RSN - 404" />
		<${Meta} k="description" v="404 - Page not found." />

		<h2 style=${css}>The page you were looking for is not on this server.</h2>
		<p style=${css}>You may <a href="/">return to the main page</a> or select a destination from the links in the header.</p>
	`
}

export default E404
