import { DEFAULT_THEME, PREFERENCES_STORAGE_KEY } from '../model/preferences';

// Applies the stored theme before first paint, so a dark-mode user never sees
// a white flash while React hydrates. Kept tiny and dependency-free because it
// runs inline in <head>; PreferencesProvider owns every later update.
const script = `
try {
  var raw = localStorage.getItem('${PREFERENCES_STORAGE_KEY}');
  var theme = raw ? (JSON.parse(raw).theme || '${DEFAULT_THEME}') : '${DEFAULT_THEME}';
  var dark = theme === 'dark' ||
    (theme === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
  if (dark) document.documentElement.classList.add('dark');
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light';
} catch (e) {}
`;

export default function ThemeScript() {
  return <script dangerouslySetInnerHTML={{ __html: script }} />;
}
