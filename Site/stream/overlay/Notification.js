
import { html, css, Component } from "/header.js"

class Notification extends Component {
	constructor(props) {
		super();

		this.state = {data: {}}
	}

	render(props, state) {
		
		return html`
			<div class="${this.css.body}" style="display: ${state.data.Type == undefined ? "none" : "block"}">
				<div class="inner">
					${(() => {
						if (state.data.Type == undefined) {
							return ""
						}
						let data = JSON.parse(state.data.Data)

						new Audio("/stream/assets/ding.mp3").play();
						switch (state.data.Type) {
						case "sub":
							switch (data.Months) {
							case 0:
								return html`
									<h2>Thank you ${data.Name}</h2>
									<p>A shiny new subscriber!</p>
								`
							case 1:
								return html`
									<h2>Thank you ${data.Name}</h2>
									<p>Subscriber for a whole month!</p>
								`
							default:
								return html`
									<h2>Thank you ${data.Name}</h2>
									<p>Subscriber for ${data.Months} months!</p>
								`
							}
						case "gift":
							return html`
								<h2>Thank you ${data.Name}</h2>
								<p>for gifting ${data.Count} subscriptions!</p>
							`
						case "bits":
							return html`
								<h2>Thank you ${data.Name}</h2>
								<p>for the ${data.Bits} bits!</p>
							`
						case "follow":
							return html`
								<h2>Thank you ${data.Name}</h2>
								<p>for the follow!</p>
							`
						case "raid":
							return html`
								<h2>Thank you ${data.Name}</h2>
								<p>for raiding with ${data.Viewers} viewers!</p>
							`
						}
					})()}
				</div>
			</div>
		`
	}

	Update(data) {
		if (data == null || data == undefined) {
			data = {}
		}
		this.setState({data: data})
	}

	css = {
		body: css`
			height: 100%;

			border-style: solid;
			border-color: var(--secondary-color);
			border-radius: 25px;
			border-width: 10px;
			background-color: var(--bg-color);

			.inner {
				width: 100%;
			}

			h2, p {
				text-align: center;
			}
			h2 {
				font-size: 2.5em;
			}
			p {
				font-size: 1.5em;
			}
		`
	}
}

export default Notification
