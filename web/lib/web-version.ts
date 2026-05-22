// Web app version pulled from package.json at build time.
// tsconfig has resolveJsonModule enabled so the import below is type-safe
// and tree-shakes down to a constant string in the bundle.
import pkg from '../package.json';

export const WEB_VERSION: string = pkg.version;
