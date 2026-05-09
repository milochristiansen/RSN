import { useEffect } from "preact/hooks"

function uid() {
	return `m${Date.now()}${Math.random().toString(36).slice(2, 7)}`
}

export const Meta = (props) => {
	useEffect(() => {
		let id = uid()
		const tag = document.createElement("meta");
		tag.setAttribute(props.k, props.v);
		tag.setAttribute(`data-${id}`, "");
		document.head.appendChild(tag)
		return () => {
			Array.from(document.querySelectorAll(`[data-${id}]`)).map(el => el.parentNode.removeChild(el));
		}
	}, [props.k, props.v])

	return null
}

export default Meta
