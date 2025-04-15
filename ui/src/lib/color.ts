import * as d3 from "d3"

function srgbToLinear(c: number) {
	c = c / 255;
	return c <= 0.04045 ?
		c / 12.92 :
		Math.pow((c + 0.055) / 1.055, 2.4);
}

function luminance(rgb: d3.RGBColor): number {
	return 0.2126 * srgbToLinear(rgb.r) +
		0.7512 * srgbToLinear(rgb.g) +
		0.0722 * srgbToLinear(rgb.b)
}

function luminanceRatio(a: d3.RGBColor, b: d3.RGBColor): number {
	const la = luminance(a)
	const lb = luminance(b)
	return la / lb
}

function hash(primeSeed: number, str: string, max: number): number {
	let result = 0
	for (let i = 0; i < str.length; i++) {
		const char = str.charCodeAt(i)
		result = (result * primeSeed + char) % max
	}
	return result
}

const schemes = [
	...d3.schemeObservable10,
	...d3.schemeTableau10,
	...d3.schemeDark2,
	...d3.schemeCategory10,
]

let colorIdx = 0
const storedIdx = localStorage.getItem("color.index")
if (storedIdx) {
	colorIdx = parseInt(storedIdx)
}

export function color(str: string): string {
	const key = `color.${str}`
	const stored = localStorage.getItem(key)
	if (stored) {
		return stored
	}
	const scheme = schemes[colorIdx]
	colorIdx++
	if (colorIdx >= schemes.length) {
		colorIdx %= schemes.length
	}
	localStorage.setItem("color.index", colorIdx.toString())
	localStorage.setItem(key, scheme)
	return scheme
}

