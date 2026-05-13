import { html, Meta, Title } from "/header.js"
import { EmailLink } from "/components/EmailLink.js"

export const Main = (props) => {
	return html`
		<${Title} text="RSN - Home" />
		<${Meta} k="description" v="Really Simple Notifier home page." />
		<h2>Welcome to RSN!</h2>
		<p>
			Really Simple Notifier is, as you may have guessed from the name, a really simple RSS notification platform.
		</p>
		<p>
			This is not an RSS reader. Instead of showing you the feed content the title and URL are extracted and
			turned into a link to take you to the actual content. Why read a simplified version when you can just go
			to the real thing?
		</p>
		<p>
			Unlike many RSS readers RSN does not waste your device resources (battery for example) constantly pulling
			new copies of all your feeds looking for updates. Instead my server does it for you. :P The server is
			constantly walking the list of feeds keeping track of which ones have new content. When you want to see
			if you have new stuff, you can check one place and get a list of everything that is new! When you are
			actively using the site it will poll the server every little bit, resulting in you seeing your new
			content within minutes of it being posted. Best of all, when you read something all your devices will
			know you read it, unlike a traditional RSS app where each instance is independent.
		</p>
		<p>
			Content is tracked by URL. If the URL of old content changes it is assumed to be a new (thankfully sites
			that do this are rare, otherwise I would have to hash the article contents or something like that). The
			server has an elephant's memory, it will never forget what you have read, so if you are subbed to one of
			those annoying sites that list every single page on the whole site in their feed you won't get bombed with
			hundreds of "new" articles every month when the RSS service throws away "stale content" (a purely
			hypothetical situation that had nothing to do with me making this in the first place. Yup, purely
			hypothetical).
		</p>
		<p>
			If you want to get push notifications for new content, that is now possible. There is an 
			${' '}<a href="/push">experimental settings page</a> that will allow you to subscribe, unsubscribe,
			and view information about your subscription. If you choose to subscribe, you will need to allow
			notifications when prompted. Don't worry, if you decide you don't like it you can always go back and
			unsubscribe. This is a relatively new feature, and feedback is appreciated.
		</p>
		<p>
			This project was made for my personal use. I use it every day for all my reading needs, so it
			${' '}<i>should</i> be pretty reliable. That said, sometimes stuff breaks. If it does use
			<${EmailLink}>this handy little email link<//> to let me know what is wrong. I'll have it working again
			shortly. This isn't the product of some big corporation, its all one dude who likes webfiction and comics
			and couldn't find a good way to handle RSS feeds across devices that didn't have a subscription fee and/or
			missing required features. Usually both.
		</p>
		<p>
			Anyway, if you want to give my little project a try you can
			${' '}<a href="/login-landing">sign up</a> at any time.
		</p>
		<p>
			If you are one of those light-mode weirdos click the sun in the bottom right corner. Pardon me, I'll be
			over there commiserating with your eyes.
		</p>
	`
}

export default Main
