
import { Component } from "preact"
import { nanoid } from "nanoid"

// This is like the Meta component, except instead of swapping tags it changes the document title.
class Title extends Component {
	state = { old: "" }

	componentDidMount() {
		this.setState({ old: document.title })
		document.title = this.props.text
	}

	componentWillUnmount() {
		document.title = this.state.old
	}
}

export default Title
