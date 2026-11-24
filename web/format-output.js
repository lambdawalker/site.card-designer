async function formatCode(source, parser) {
  const [{ format }, plugin] = await Promise.all([
    import('prettier/standalone'),
    parser === 'html' ? import('prettier/plugins/html') : import('prettier/plugins/postcss'),
  ]);
  return format(source, { parser, plugins: [plugin], embeddedLanguageFormatting: 'off' });
}

export async function formatOutput(source, parser) {
  try {
    return await formatCode(source, parser);
  } catch {
    return source;
  }
}

// Keep Jinja tokens byte-for-byte intact while formatting their surrounding code.
export async function formatSource(source, parser) {
  let prefix = 'jinja_format_';
  while (source.toLowerCase().includes(prefix)) prefix += '_';
  const tokens = [];
  const masked = source.replace(/{{(?:[^'"]|'(?:\\.|[^'\\])*'|"(?:\\.|[^"\\])*")*?}}|{%(?:[^'"]|'(?:\\.|[^'\\])*'|"(?:\\.|[^"\\])*")*?%}|{#[\s\S]*?#}/g, token => {
    const marker = prefix + tokens.length + '_end';
    const placeholder = token.startsWith('{{')
      ? marker
      : (parser === 'css' ? `/* ${marker} */` : `<!--${marker}-->`);
    tokens.push({ placeholder, token });
    return placeholder;
  });
  let formatted = await formatCode(masked, parser);
  for (const { placeholder, token } of tokens) {
    if (!formatted.includes(placeholder)) throw Error('Could not preserve a template expression during formatting.');
    formatted = formatted.split(placeholder).join(token);
  }
  return formatted;
}
