import { useState, useCallback, useEffect } from "preact/hooks"
import { html } from "/header.js"
import { createContext } from "preact"

export const AuthContext = createContext({ok: false, refresh: () => {}, whoami: null})

// Alias to make the API orthogonal
export const AuthConsumer = AuthContext.Consumer

export function AuthProvider(props) {
	let whoamiR = localStorage.getItem('whoami')
	let initialAuth
	if (whoamiR != "") {
		initialAuth = {ok: true, refresh: refresh, whoami: JSON.parse(whoamiR)}
	} else {
		initialAuth = {ok: false, refresh: refresh, whoami: null}
	}

	let [auth, setAuth] = useState(initialAuth)

	function refresh() {
		fetch("/auth/whoami", {
			credentials: 'include'
		})
			.then(r => {
				if (!r.ok) {
					if (r.status != 403) {
						setTimeout(() => refresh(), 5000)
						throw new Error("Request failed, retrying in 5s.")
					}
					setAuth({ok: false, refresh: refresh, whoami: null})
					localStorage.setItem('whoami', "")
					throw new Error("Login invalid.")
				}
				return r.json()
			})
			.then(whoami => {
				setAuth({ok: true, refresh: refresh, whoami: whoami})
				localStorage.setItem('whoami', JSON.stringify(whoami))
			})
	}

	useEffect(() => {
		refresh()
	}, [])

	return html`
		<${AuthContext.Provider} value=${auth}>
			${props.children}
		<//>
	`
}
