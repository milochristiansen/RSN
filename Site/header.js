
// This file is intended to be imported in all components, and provides all the basic dependencies 

// Import my HTML and CSS processors
import htm from "htm"
import css from "csz"

// Import PReact
import { h, render, Component, createRef, createContext } from "preact"

// Enable debugging. Comment this out for production.
import "preact/debug"

// Bind PReact to the HTML processor
const html = htm.bind(h)

// These components allow you to modify the document title and meta tags on a per route basis.
import Meta from "/components/Meta.js"
import Title from "/components/Title.js"

// Export all that stuff so the components can use it.
export { html, css, h, render, Component, createRef, createContext, Meta, Title }
