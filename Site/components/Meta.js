
import { Component } from "preact"
import { nanoid } from "nanoid"

// This "Component" just adds a meta tag to the header of any page it is included on. When the component is unmounted,
// the tag is removed.
class Meta extends Component {
	constructor() {
		super();
		this.state = { id: nanoid(5) };
	}

	componentDidMount() {
		const tag = document.createElement("meta");
		tag.setAttribute(this.props.k, this.props.v);
		tag.setAttribute(`data-${this.state.id}`, "");
		document.head.appendChild(tag)
	}

	componentWillUnmount() {
		Array.from(document.querySelectorAll(`[data-${this.state.id}]`)).map(el => el.parentNode.removeChild(el));
	}
}

export default Meta
