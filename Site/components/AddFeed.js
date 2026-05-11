import { useState, useCallback } from "preact/hooks"
import { html } from "/header.js"
import { Style } from "/components/Style.js"

const StatusMessage = Style.span`
	&.error {
		color: var(--warning-color);
	}
`

const Status = Style.div`

`

const AddBody = Style.section`
	margin-top: 5px;
	margin-bottom: 10px;

	margin-left: 10px;
	margin-right: 10px;
`

const AddForm = Style.form`
	display: flex;
	flex-direction: column;

	input {
		margin-top: 2px;
	}
`

export const AddFeed = (props) => {
	let [addState, setAddState] = useState(null)
	let [url, setUrl] = useState("")
	let [name, setName] = useState("")

	let addfeed = useCallback((evnt) => {
		evnt.preventDefault()

		if (url == "" || name == "") {
			setAddState(false)
			setTimeout(() => setAddState(null), 5000);
			return;
		}

		fetch("/api/feeds", {
			method: "POST",
			credentials: "include",
			body: JSON.stringify({
				URL: String(url),
				Name: String(name)
			})
		})
			.then((res) => {
				if (res.ok) {
					setAddState(true)
					setUrl("")
					setName("")
					setTimeout(() => setAddState(null), 3000);
					return;
				}
				throw new Error(res.status);
			})
			.catch(error => {
				console.error(error.message);
				setAddState(false)
				setTimeout(() => setAddState(null), 5000);
			});
	}, [url, name])

	let handleInput = useCallback((e) => {
		if (e.target.name === "url") {
			setUrl(e.target.value)
		} else if (e.target.name === "name") {
			setName(e.target.value)
		}
	}, [])

	let status = html`<${StatusMessage}>${"\u200b"}<//>`
	if (addState === false) {
		status = html`<${StatusMessage} class="error">Failed adding feed.<//>`
	} else if (addState === true) {
		status = html`<${StatusMessage}>Feed added!<//>`
	}

	return html`
		<${AddBody}>
			<${Status}>
				${status}
			<//>
			<${AddForm} onsubmit=${addfeed}>
				<input type="text" placeholder="Feed URL" name="url" value=${url} onInput=${handleInput} />
				<input type="text" placeholder="Feed Name" name="name" value=${name} onInput=${handleInput} />
				<input type="submit" value="Subscribe Feed" />
			<//>
		<//>
	`
}

export default AddFeed
