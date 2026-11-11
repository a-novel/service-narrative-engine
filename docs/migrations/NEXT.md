# Upgrading to NEXT

NEXT accepts only project tags in lower kebab case, such as `huis-clos` or `années-50`, and refuses any other tag.

## Do I need to act?

**Required** for a client that sends project tags: a tag outside lower kebab case now answers 400 on create and on update. A client that sends no tags, or only reads projects, keeps working. Genres are unchanged, and the release migrates no data.

## Send tags in lower kebab case

A tag is one or more runs of lowercase letters, marks or digits, joined by single hyphens. It holds no whitespace, no uppercase letter and no underscore, and no hyphen at either end or twice in a row. Letters come from any script, with the marks and modifier letters their words carry, so French tags keep their accents and Thai, Hindi or Japanese words stay whole. The limits stay: at most 10 tags of 1 to 24 characters, each appearing once.

| Tag              | Accepted |
| ---------------- | -------- |
| `annees-50`      | yes      |
| `années-50`      | yes      |
| `1950`           | yes      |
| `ラーメン`       | yes      |
| `Années 50`      | no       |
| `roman_maritime` | no       |
| `huis--clos`     | no       |
| `-noir`          | no       |

The service refuses a tag and never rewrites it, so normalize tags as the writer types them:

```diff
 await projectCreate(api, accessToken, {
   title,
   language,
-  tags: ["Huis clos", "roman_maritime"],
+  tags: ["huis-clos", "roman-maritime"],
 });
```

Write accented letters precomposed, as `normalize("NFC")` returns them. A decomposed accent, an `e` followed by a combining mark, is refused.

`ProjectTagSchema` in the JS client checks the same rule, so `ProjectTagsSchema.safeParse(tags).success` tells whether the service accepts a list. A refused tag names the `projecttag` rule in the problem's `invalidFields`, as in `{"tags[0]": "projecttag"}`. A tag with whitespace at either end named `projectlabel` before.

## Notes

A project keeps any tag it stored before this release, and its next update must send the list in lower kebab case. No client wrote tags before this release, so no stored tag is expected to break.
