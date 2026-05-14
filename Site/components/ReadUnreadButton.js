import { useState, useRef, useCallback } from "preact/hooks"
import { html } from "/header.js"
import { Style } from "/components/Style.js"

export const OnRead = new Event('read', {bubbles: true});
export const OnUnread = new Event('unread', {bubbles: true});

const BtnContainer = Style.a`
	height: 32px;
	width: 32px;
	position: relative;

	margin-top: auto;
	margin-bottom: auto;

	&.on {
		--color: var(--on-color);
	}
	&.off {
		--color: var(--off-color);
	}
	&.confirm {
		--color: var(--warning-color);
	}
`

const Checkmark = Style.div`
	position: absolute;
	left: 10px;
	top: 2px;

	display: inline-block;
	transform: rotate(45deg);
	height: 24px;
	width: 10px;
	border-bottom: 5px solid var(--color);
	border-right: 5px solid var(--color);
`

export const ReadUnreadButton = (props) => {
	let [buttonstate, setButtonstate] = useState(false)
	let root = useRef()

	let doclick = (evnt) => {
		evnt.preventDefault()

		setButtonstate(state => {
			if (state) {
				if (props.state) {
					if (props.aid != undefined) {
						fetch("/api/articles/" + props.aid, {
							method: 'PATCH',
							credentials: 'include',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify({ read: false })
						}).then(r => {
							if (r.ok) {
								console.log(root.current, root.current.dispatchEvent)
								root.current.dispatchEvent(OnUnread)
							}
						})
					}
				} else {
					if (props.aid != undefined) {
						fetch("/api/articles/" + props.aid, {
							method: 'PATCH',
							credentials: 'include',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify({ read: true })
						}).then(r => {
							if (r.ok) {
								console.log(root.current, root.current.dispatchEvent)
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
	}

	let cls = "off"
	if (props.state) {
		cls = "on"
	}
	if (buttonstate) {
		cls = "confirm"
	}

	return html`
		<${BtnContainer} ref=${root} href="/toggle-read" class=${cls} onclick=${(e) => doclick(e)} target="_top"><${Checkmark}><//><//>
	`
}

export default ReadUnreadButton
