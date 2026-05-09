import { useState, useRef, useCallback } from "preact/hooks"
import { html } from "/header.js"

const OnRead = new Event('read', {bubbles: true});
const OnUnread = new Event('unread', {bubbles: true});

let modesCss = {
	on: `--color: var(--on-color)`,
	off: `--color: var(--off-color)`,
	confirm: `--color: var(--warning-color)`,
}

let bodyCss = `
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
`

export const ReadUnreadButton = (props) => {
	let [buttonstate, setButtonstate] = useState(false)
	let root = useRef()

	let doclick = useCallback((evnt) => {
		evnt.preventDefault()

		setButtonstate(state => {
			if (state) {
				if (props.state) {
					if (props.aid != undefined) {
						fetch("/api/article/unread?id=" + props.aid).then(r => {
							if (r.ok) {
								root.current.dispatchEvent(OnUnread)
							}
						})
					}
				} else {
					if (props.aid != undefined) {
						fetch("/api/article/read?id=" + props.aid).then(r => {
							if (r.ok) {
								root.current.dispatchEvent(OnRead)
							}
						})
					}
				}
				return false
			}

			setTimeout(() => setButtonstate(false), 2500);
			return true
		})
	}, [props.state, props.aid])

	let cls = modesCss.off
	if (props.state) {
		cls = modesCss.on
	}
	if (buttonstate) {
		cls = modesCss.confirm
	}

	return html`
		<a ref=${root} href="/toggle-read" class=${[cls, bodyCss].join(" ")} onclick=${(e) => doclick(e)} native><div></div></a>
	`
}

export { ReadUnreadButton, OnRead, OnUnread }
export default ReadUnreadButton
