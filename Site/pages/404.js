
import { html, css, Meta, Title, Component } from "/header.js"

class E404 extends Component {
	render(props, state) {
		return html`
			<${Title} text="RSN - 404" />
			<${Meta} k="description" v="404 - Page not found." />

			<h2 class=${this.css.all}>The page you were looking for is not on this server.</h2>
			<p class=${this.css.all}>You may <a href="/">return to the main page</a> or select a destination from the links in the header.</p>
		`;
	}

	css = {
		all: css`
			width: 100%;
			text-align: center;
		`
	}
}

export default E404
