import { useState, useCallback } from "preact/hooks"
import { html } from "/header.js"
import { createContext } from "preact"

export const ThemeContext = createContext({refresh: () => {}, theme: "dark"})

// Alias to make the API orthogonal
export const ThemeConsumer = ThemeContext.Consumer

export function ThemeProvider(props) {
	let theme = localStorage.getItem('theme') == "light" ? "light" : "dark"
	document.documentElement.setAttribute("data-theme", theme);

	let [data, setData] = useState({theme: theme})

	let toggle = useCallback((evnt) => {
		setData(state => {
			let theme = state.theme == "light" ? "dark" : "light"
			document.documentElement.setAttribute("data-theme", theme);
			localStorage.setItem("theme", theme)
			return {theme: theme}
		})
		evnt.preventDefault();
	}, [])

	return html`
		<${ThemeContext.Provider} value=${{toggle, theme: data.theme}}>
			${props.children}
		<//>
	`
}
