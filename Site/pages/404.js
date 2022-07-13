
import { html, css, Meta, Title, Component } from "/header.js"

class E404 extends Component {
	render(props, state) {
		return html`
			<${Title} text="RSN - 404" />
			<${Meta} k="description" v="404 - Page not found." />
			<p>The page you were looking for is not on this server.</p>
		`;
	}
}

export default E404
