import { useEffect, useState } from "preact/hooks"

export const Title = (props) => {
	let [old, setOld] = useState("")

	useEffect(() => {
		setOld(document.title)
		document.title = props.text
		return () => { document.title = old }
	}, [props.text])

	return null
}

export default Title
