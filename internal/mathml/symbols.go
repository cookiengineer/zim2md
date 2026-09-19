package mathml

// operatorClass mirrors the MathML operator spacing classes (the ones MathJax
// records in the data-mjx-texclass attribute).
type operatorClass int

const (
	classOrd operatorClass = iota
	classBin
	classRel
	classOpen
	classClose
	classPunct
	classInner
)

// mathVariantCommands maps a MathML mathvariant to the matching LaTeX command.
// Variants that need no wrapper (the default italic) are intentionally absent.
var mathVariantCommands = map[string]string{
	"normal":          `\mathrm`,
	"bold":            `\mathbf`,
	"bold-italic":     `\boldsymbol`,
	"double-struck":   `\mathbb`,
	"fraktur":         `\mathfrak`,
	"bold-fraktur":    `\mathfrak`,
	"script":          `\mathcal`,
	"bold-script":     `\mathcal`,
	"initial":         `\mathcal`,
	"tailed":          `\mathcal`,
	"looped":          `\mathcal`,
	"stretched":       `\mathcal`,
	"sans-serif":      `\mathsf`,
	"bold-sans-serif": `\mathsf`,
	"monospace":       `\mathtt`,
}

// runeCommands maps a single Unicode rune to a LaTeX command.
var runeCommands = map[rune]string{
	// Lower-case Greek.
	'α': `\alpha`, 'β': `\beta`, 'γ': `\gamma`, 'δ': `\delta`,
	'ε': `\epsilon`, 'ζ': `\zeta`, 'η': `\eta`, 'θ': `\theta`,
	'ι': `\iota`, 'κ': `\kappa`, 'λ': `\lambda`, 'μ': `\mu`,
	'ν': `\nu`, 'ξ': `\xi`, 'π': `\pi`, 'ρ': `\rho`,
	'σ': `\sigma`, 'ς': `\varsigma`, 'τ': `\tau`, 'υ': `\upsilon`,
	'φ': `\phi`, 'χ': `\chi`, 'ψ': `\psi`, 'ω': `\omega`,
	'ϑ': `\vartheta`, 'ϕ': `\varphi`, 'ϖ': `\varpi`, 'ϱ': `\varrho`,
	// Upper-case Greek.
	'Γ': `\Gamma`, 'Δ': `\Delta`, 'Θ': `\Theta`, 'Λ': `\Lambda`,
	'Ξ': `\Xi`, 'Π': `\Pi`, 'Σ': `\Sigma`, 'Υ': `\Upsilon`,
	'Φ': `\Phi`, 'Ψ': `\Psi`, 'Ω': `\Omega`,
	// Blackboard bold sets.
	'ℕ': `\mathbb{N}`, 'ℤ': `\mathbb{Z}`, 'ℚ': `\mathbb{Q}`,
	'ℝ': `\mathbb{R}`, 'ℂ': `\mathbb{C}`, 'ℙ': `\mathbb{P}`,
	'ℍ': `\mathbb{H}`, '𝔽': `\mathbb{F}`, '𝔼': `\mathbb{E}`,
	// Miscellaneous symbols that commonly appear as identifiers.
	'∞': `\infty`, '∂': `\partial`, '∇': `\nabla`, '∅': `\emptyset`,
	'ℓ': `\ell`, 'ℏ': `\hbar`, 'ℜ': `\Re`, 'ℑ': `\Im`,
	'ℵ': `\aleph`, '◻': `\square`, '△': `\triangle`, '∠': `\angle`,
	'′': `'`, '″': `''`, '‴': `'''`,
}

// functionNames maps multi-letter identifiers to the matching LaTeX operator.
var functionNames = map[string]string{
	"sin": `\sin`, "cos": `\cos`, "tan": `\tan`, "cot": `\cot`,
	"sec": `\sec`, "csc": `\csc`, "arcsin": `\arcsin`, "arccos": `\arccos`,
	"arctan": `\arctan`, "sinh": `\sinh`, "cosh": `\cosh`, "tanh": `\tanh`,
	"coth": `\coth`, "log": `\log`, "ln": `\ln`, "exp": `\exp`,
	"lim": `\lim`, "limsup": `\limsup`, "liminf": `\liminf`,
	"max": `\max`, "min": `\min`, "sup": `\sup`, "inf": `\inf`,
	"deg": `\deg`, "det": `\det`, "dim": `\dim`, "gcd": `\gcd`,
	"hom": `\hom`, "ker": `\ker`, "arg": `\arg`, "Pr": `\Pr`,
	"mod": `\bmod`, "bmod": `\bmod`,
}

// accentCommands maps an <mover> accent glyph to the matching LaTeX command.
var accentCommands = map[string]string{
	"^": `\hat`, "ˆ": `\hat`,
	"¯": `\bar`, "‾": `\bar`,
	"→": `\vec`, "⇀": `\vec`,
	"˙": `\dot`, "¨": `\ddot`,
	"~": `\tilde`, "˜": `\tilde`,
	"⏞": `\overbrace`, "⏟": `\underbrace`,
	"←": `\overleftarrow`, "↔": `\overleftrightarrow`,
}

// operatorCommands maps an <= 2 rune MathML operator sequence to LaTeX.
var operatorCommands = map[string]string{
	"−": "-", "–": "-", "—": "-",
	"±": `\pm`, "∓": `\mp`, "×": `\times`, "÷": `\div`,
	"⋅": `\cdot`, "·": `\cdot`, "∙": `\cdot`, "∗": `\ast`, "∘": `\circ`,
	"≤": `\leq`, "≥": `\geq`, "≠": `\neq`, "≈": `\approx`,
	"≡": `\equiv`, "∼": `\sim`, "≅": `\cong`, "∝": `\propto`,
	"∈": `\in`, "∉": `\notin`, "∋": `\ni`, "⊂": `\subset`,
	"⊃": `\supset`, "⊆": `\subseteq`, "⊇": `\supseteq`,
	"∪": `\cup`, "∩": `\cap`, "∖": `\setminus`, "∅": `\emptyset`,
	"∑": `\sum`, "∏": `\prod`, "∫": `\int`, "∮": `\oint`,
	"⋃": `\bigcup`, "⋂": `\bigcap`, "⨁": `\bigoplus`, "⨂": `\bigotimes`,
	"→": `\to`, "←": `\leftarrow`, "↔": `\leftrightarrow`,
	"⟶": `\longrightarrow`, "⟵": `\longleftarrow`, "⟼": `\longmapsto`,
	"↦": `\mapsto`, "⇒": `\Rightarrow`, "⇐": `\Leftarrow`,
	"⇔": `\Leftrightarrow`, "∀": `\forall`, "∃": `\exists`,
	"¬": `\neg`, "∧": `\wedge`, "∨": `\vee`, "⊥": `\perp`,
	"⋯": `\cdots`, "…": `\ldots`, "⋮": `\vdots`, "⋱": `\ddots`,
	"∞": `\infty`, "∂": `\partial`, "∇": `\nabla`,
	"′": `'`, "″": `''`, "‴": `'''`,
	"|": `|`, "‖": `\|`,
}

// operatorClasses maps an operator sequence to its spacing class.
var operatorClasses = map[string]operatorClass{
	"+": classBin, "-": classBin, "−": classBin, "±": classBin, "∓": classBin,
	"×": classBin, "÷": classBin, "⋅": classBin, "·": classBin, "∙": classBin,
	"∗": classBin, "∘": classBin, "∪": classBin, "∩": classBin,
	"∧": classBin, "∨": classBin,
	"=": classRel, "≠": classRel, "≤": classRel, "≥": classRel, "≈": classRel,
	"≡": classRel, "∼": classRel, "≅": classRel, "∝": classRel, "∈": classRel,
	"∉": classRel, "∋": classRel, "⊂": classRel, "⊃": classRel, "⊆": classRel,
	"⊇": classRel, "→": classRel, "←": classRel, "↔": classRel, "⟶": classRel,
	"⟵": classRel, "⟼": classRel, "↦": classRel, "⇒": classRel, "⇐": classRel,
	"⇔": classRel, "⊥": classRel, "⋮": classRel, "⋱": classRel,
	"⋯": classRel, "…": classRel,
	"(": classOpen, "[": classOpen, "{": classOpen,
	"⟨": classOpen, "⌊": classOpen, "⌈": classOpen,
	")": classClose, "]": classClose, "}": classClose,
	"⟩": classClose, "⌋": classClose, "⌉": classClose,
	",": classPunct, ";": classPunct, ":": classPunct,
}
