
import { html, css, Meta, Title, Component } from "/header.js"

class LoginLanding extends Component {
	render(props, state) {
		return html`
			<${Title} text="RSN - Login/Sign-up" />
			<${Meta} k="description" v="Really Simple Notifier login and signup page." />
			<h2>Login and Account Information</h2>
			<p>
				TLDR: Use the link in the header dingus.
			</p>
			<p>
				Hello new or returning user! You probably got here because you want to create a new account. Well,
				good news! Unlike pretty much every other annoying site on the internet I don't care about who you are,
				when you were born, what your address is, what size shoes you wear, etc. I only want one thing: Some
				kind of unique identifier that I can use to specify what data in the system is yours and what isn't.
			</p>
			<p>
				Rather than go to a lot of work making a password login system, I just let Google do it for me and then
				ask them to give me a number that identifies you uniquely. Due to the way this third-party login system
				works the least intrusive information I can ask for about you is your email address, so that is all I
				will request access to (and I won't store it anywhere).
			</p>
			<p>
				Actually, part of me does want to store your email somewhere. What if something goes wrong with the
				site that I need to notify users about? How will I contact people if I have to? So far it hasn't
				been needed, and so I don't keep emails hoping it never will. You're welcome.
			</p>
			<p>
				Anyway, since I don't store anything about you in the system except that ID number I get, there isn't
				any need to create an account. Just login and if I don't know you I'll store anything you do using your
				number, and if I do know you I'll just show you the stuff owned by your number.
			</p>
			<p>
				That means you just <strong><a href="/auth/login/google" native>Login with Google</a></strong> and go
				have fun!
			</p>
			<p>
				PS: If you want to delete your "account" or export your data I can arrange for that to be possible, just
				no one has needed it yet so I haven't written it as a feature. Hit me up at the email address in the footer.
			</p>
		`;
	}

	css = {
	}
}

export default LoginLanding
