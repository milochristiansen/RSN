 
import { html, css, Component } from "/header.js"
import { route } from 'preact-router';

import { AuthContext } from '/components/Auth.js'

// AuthedCommonStructure handles the mechanics of getting the auth state for you. Unlike Component, your
// component should not define render, but rather override renderAuthed instead.
class AuthedCommonStructure extends Component {
	static contextType = AuthContext

	render(props, state) {
		if (this.maybeRedirect()) {
			return null
		}
		return this.renderAuthed(this.context, props, state)
	}

	// Override this instead of render.
	renderAuthed(auth, props, state) {
		return html`<p>Someone goofed.</p>`
	};

	componentWillMount() {
		this.maybeRedirect()
	}

	// If you want to just straight-up redirect away if not authorized, override this to return a URL that should be
	// used as a redirect target. This redirect can happen right away if the cached auth data was already invalid, or
	// some time after the user loads the page if the cached data was valid but the refreshed data is not. This doesn't
	// really protect anything! This is not a security measure!
	noAuthRedirect() {
		return null
	}

	// If auth state is invalid and there is a redirect URL, redirect and return true. Otherwise return false.
	maybeRedirect() {
		if (!this.context.ok) {
			const to = this.noAuthRedirect()
			if (to == "" || to == null || to == undefined) {
				return false
			}
			route(to, true)
			return true
		}
		return false
	}
}

export default AuthedCommonStructure
