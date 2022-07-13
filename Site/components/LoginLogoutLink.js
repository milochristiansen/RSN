
import { html } from "/header.js"
import { AuthContext } from "/components/Auth.js"
import { useContext } from "preact/hooks"

function LoginLogoutLink(props) {
	let auth = useContext(AuthContext)

	if (auth.ok) {
		return html`<a href="/auth/logout" native>Logout</a>`
	}
	return html`<a href="/auth/login/google" native>Login</a>`
}

export default LoginLogoutLink
