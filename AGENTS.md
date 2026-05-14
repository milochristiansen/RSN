

## Style Guide:

* The project domain name is `httpscolonslashslashwww.com`, I know it looks like a place holder, but that is the real domain.
* It is a convention in this project that all Go programs use files named `keys.go` to store any secrets that will be compiled into the binary.
	* During reviews, always check to see if this file is actually committed before mentioning it as a security issue.
* Do not forget to use HTM style closing tags (`<${component}><//>`) instead of JSX style closing tags (`<${component}></${component}>`) for non-self-closing HTML tags.
	* Any time you do an action that edits HTML or javascript, you should check to make sure you used the correct HTM closing tag(s) afterwards.
* All component files should export the main component as the default export and also as a named export.
* Named exports in component files should be inline with the declaration where possible (eg. `export const Xyz = "thing"` instead of `const Xyz = "thing"; export { Xyz }`).
* Where possible all imports should use named imports (`import { Xyz } from "file"` vs `import Xyz from "file"`).
* All components should be defined as const arrow functions, and any handlers defined inside these components should also be defined this way.
* When working with Go code, all errors MUST be handled. Never call a function that can return `error` without checking that error and either returning it to the parent caller or logging it.
