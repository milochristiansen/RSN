import { useEffect } from "preact/hooks"
import { route } from 'preact-router';

import { AuthContext } from '/components/Auth.js'
import { useContext } from "preact/hooks"

// useAuthRedirect checks authentication state and redirects to path if the state is not valid.
export const useAuthRedirect = (path = "/") => {
	let auth = useContext(AuthContext)

	useEffect(() => {
		if (!auth.ok && path) {
			route(path)
		}
	}, [auth.ok, path])
}

export default useAuthRedirect
