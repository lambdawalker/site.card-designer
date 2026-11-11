import {generateTheme} from '@lambdawalker/drawing-tools/draw/color/theme.ts';

// Resolve before preview/export as well as UI updates so seed-driven themes replay.
export function themedDesign(design) {
  if (!design.auto_theme) return design;
  const theme = generateTheme(design.seed);
  return {...design, color: theme.primary_color, text_color: theme.text_color};
}
