import { html } from "/header.js"
import { Style } from "/components/Style.js"
import { AuthContext } from "/components/Auth.js"
import { useContext } from "preact/hooks"

const LoginLogoutLink = (props) => {
	let auth = useContext(AuthContext)
	let Link = props.as ?? Style.a

	if (auth.ok) {
		return html`<${Link} href="/auth/logout" native>Logout<//>`
	}
	return html`<${Link} href="/auth/login/google" native>Login<//>`
}

export { LoginLogoutLink }
export default LoginLogoutLink
