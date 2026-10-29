# Manuscript format v1

A manuscript is the text of a narrative, exported from one manuscript layer of a project by `GET /v1/manuscript`. It is the raw interchange format: a renderer reads it and produces a consumer format, such as a novel, a screenplay or a game. A project may hold several manuscript layers, such as a branch of the story, and each exports as its own manuscript.

The JSON Schema [`manuscript.schema.json`](../internal/models/schemas/manuscript.schema.json) defines the format, and every export follows it. The REST client exports the same definition as `ManuscriptSchema`.

## Document

| Field           | Content                                                                         |
| --------------- | ------------------------------------------------------------------------------- |
| `format`        | Always `manuscript`.                                                            |
| `formatVersion` | Always `1`.                                                                     |
| `title`         | The title of the project.                                                       |
| `language`      | The language of the text, as a BCP 47 tag: `en` or `fr` today.                  |
| `scenes`        | Every scene of the project, in narrative order, each with `title` and `blocks`. |
| `appendices`    | Data about the text, keyed by appendix name. Format v1 defines none.            |

A scene the layer holds no text for keeps its place with `"blocks": []`, so the structure of the narrative survives the export.

## Blocks

A scene's blocks come in reading order. [`block.schema.json`](../internal/models/schemas/block.schema.json) defines their two kinds:

- `narration` holds prose the narrator tells, in `text`. A blank line separates two paragraphs.
- `dialogue` holds one line a `speaker` says, as `parts` in speaking order. A part holds either `text`, the words spoken, or `tone`, how the speaker says the words that follow.

Blocks carry no ID. An appendix names a block by the position of its scene and its own position in that scene, both from 0.

## Reader rules

- A reader checks `format` and `formatVersion` first, and refuses a document it does not know. A change that a reader of version 1 cannot read raises the version.
- A reader skips an appendix it does not know.
- Typography belongs to the renderer. Dialogue parts hold the words without quotation marks or dashes, and the renderer sets them in the style of the target format and `language`, such as French guillemets and the space French puts before `?` and `!`.

## Example

```json
{
  "format": "manuscript",
  "formatVersion": 1,
  "title": "Les Heures claires",
  "language": "fr",
  "scenes": [
    {
      "title": "Ouverture",
      "blocks": [
        { "kind": "narration", "text": "La pluie battait les vitres." },
        {
          "kind": "dialogue",
          "speaker": "Marie",
          "parts": [{ "tone": "à voix basse" }, { "text": "Tu es venu." }]
        }
      ]
    },
    { "title": "Interlude", "blocks": [] }
  ],
  "appendices": {}
}
```
