
import { html, css, Component, createRef } from "/header.js"

const OnRead = new Event('read', {bubbles: true});
const OnUnread = new Event('unread', {bubbles: true});

class ReadUnreadButton extends Component {
	constructor() {
		super();
	
		this.state = {buttonstate: false}

		this.root = createRef()
	}

	doclick(evnt) {
		evnt.preventDefault()

		this.setState(state => {
			if (state.buttonstate) {
				// Do the thing.
				if (this.props.state) {
					if (this.props.aid != undefined) {
						fetch("/api/article/unread?id=" + this.props.aid).then(r => {
							if (r.ok) {
								this.root.current.dispatchEvent(OnUnread)
							}
						})
					}
				} else {
					if (this.props.aid != undefined) {
						fetch("/api/article/read?id=" + this.props.aid).then(r => {
							if (r.ok) {
								this.root.current.dispatchEvent(OnRead)
							}
						})
					}
				}
				return {buttonstate: false}
			}

			setTimeout(() => (this.setState({buttonstate: false})), 2500);
			return {buttonstate: true}
		})
	}

	render(props, state) {
		let cls = this.css.modes.off
		if (props.state) {
			cls = this.css.modes.on
		}
		if (state.buttonstate) {
			cls = this.css.modes.confirm
		}

		return html`
			<a ref=${this.root} href="/toggle-read" class=${[cls, this.css.body].join(" ")} onclick=${(e) => this.doclick(e)} native><div></div></a>
		`
	}

	css = {
		modes: {
			on: css`
				--color: var(--on-color)
			`,
			off: css`
				--color: var(--off-color)
			`,
			confirm: css`
				--color: var(--warning-color)
			`,
		},
		body: css`
			height: 32px;
			width: 32px;
			position: relative;

			margin-top: auto;
			margin-bottom: auto;

			div {
				position: absolute;
				left: 10px;
				top: 2px;

				display: inline-block;
				transform: rotate(45deg);
				height: 24px;
				width: 10px;
				border-bottom: 5px solid var(--color);
				border-right: 5px solid var(--color);
			}
		`,
	}
}

export default ReadUnreadButton

export { OnRead, OnUnread, ReadUnreadButton }
