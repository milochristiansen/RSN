
import { html, Component, createContext } from "/header.js"

const AuthContext = createContext({ok: false, refresh: () => {}, whoami: null})

// Alias to make the API orthogonal
const AuthConsumer = AuthContext.Consumer

class AuthProvider extends Component {
	constructor() {
		super();
	
		const whoamiR = localStorage.getItem('whoami')
		if (whoamiR != "") {
			this.state = {auth: {ok: true, refresh: this.refresh, whoami: JSON.parse(whoamiR)}}
		} else {
			this.state = {auth: {ok: false, refresh: this.refresh, whoami: null}}
		}

		this.refresh()
	}

	refresh() {
		fetch("/auth/whoami", {
			credentials: 'include'
		})
			.then(r => {
				if (!r.ok) {
					throw new Error("Request failed.")
				}
				return r.json()
			})
			.then(whoami => {
				this.setState({auth: {ok: true, refresh: this.refresh, whoami: whoami}})
				localStorage.setItem('whoami', JSON.stringify(whoami))
			})
			.catch(err => {
				console.log(err)
				this.setState({auth: {ok: false, refresh: this.refresh, whoami: null}})
				localStorage.setItem('whoami', "")
			})
	}

	render(props, state) {
		return html`
			<${AuthContext.Provider} value=${state.auth}>
				${props.children}
			<//>
		`
	}
}

export { AuthContext, AuthConsumer, AuthProvider }
