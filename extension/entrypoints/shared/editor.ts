import { basicSetup } from "codemirror"
import { EditorView } from "@codemirror/view"
import { languages } from "@codemirror/language-data"
import { LanguageDescription } from "@codemirror/language"

async function readFile(filepath: string) {
    const resp = await browser.runtime.sendMessage({
        jsonrpc: "2.0",
        id: crypto.randomUUID(),
        method: "readFile",
        params: { path: filepath }
    })
    return resp.result.content
}

async function main() {
    const filepath = new URLSearchParams(window.location.search).get("file")
    if (!filepath) {
        throw new Error("file parameter is required")
    }

    const filename = filepath.split("/").pop()!
    globalThis.document.title = filename ? `${filename} - Tweety` : "Tweety"

    const initialContent = await readFile(filepath)

    const extensions = [basicSetup, EditorView.lineWrapping]

    const lang = LanguageDescription.matchFilename(languages, filename)
    if (lang) {
        const langSupport = await lang.load()
        extensions.push(langSupport)
    }

    // Add an update listener to send a save message on change
    extensions.push(EditorView.updateListener.of(async (update) => {
        if (update.docChanged) {
            await browser.runtime.sendMessage({
                jsonrpc: "2.0",
                id: crypto.randomUUID(),
                method: "writeFile",
                params: {
                    path: filepath,
                    content: update.state.doc.toString()
                }
            })
        }
    }))

    extensions.push(EditorView.theme({
        "&": {
            fontSize: "11pt"
        },
    }))

    const view = new EditorView({
        doc: initialContent,
        parent: document.body,
        extensions
    })

    // Refresh file content when tab is focused
    window.addEventListener("focus", async () => {
        const newContent = await readFile(filepath)
        if (newContent !== view.state.doc.toString()) {
            view.dispatch({
                changes: { from: 0, to: view.state.doc.length, insert: newContent }
            })
        }
    })
}

main()
