
// This file is intended to be imported in all components, and provides all the basic dependencies 

// Import my HTML processors
import htm from "htm"

// Import PReact
import { h, render } from "preact"

// Bind PReact to the HTML processor
const html = htm.bind(h)

// These components allow you to modify the document title and meta tags on a per route basis.
import { Meta } from "/components/Meta.js"
import { Title } from "/components/Title.js"

// Styled components thingy.
import { Style } from "/components/Style.js"

// Export all that stuff so the components can use it.
export { html, h, render, Meta, Title, Style }
