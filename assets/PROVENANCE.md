Designer source extracted from `ds.plain_id_card`, commit `f87d1742818cbff2b04bf9c0e48b0a74b19782fc`. The original repository remains unchanged.

`catalog.json` contains the editable sources from `card-designer/ide/records/layouts/id_cards`, using `records/design-bundles/catalog.yaml` ordering. It includes A–G two variants each and the Georgia front/back pair. Source CSS includes each family stylesheet and explicit layout stylesheet. Python dictionary `common.get('key')` expressions are normalized to Nunjucks `common.key`. Identity fields without a dataset value render as an em dash. Machine codes are synthetic seeded specimens, encoded using bwip-js. Font fallback uses browser Arial; source font files were not bundled.

Drawing tools, including licenses and third-party notices, are retained under `packages/drawing-tools`.
