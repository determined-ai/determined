/**
 * Generates a four-character random hash
 * @param {object} obj
 * @param {number} [obj.length] The length of the hash
 * @returns Alphanumeric hash
 */
export const randIdAlphanumeric = ({ length = 4 }: { length?: number } = {}): string =>
  Math.random()
    .toString(36)
    .substring(2, 2 + length);

/**
 * Generates a four-character numeric hash
 * @param {object} obj
 * @param {number} [obj.length] The length of the hash
 * @returns Numeric hash
 */
export const randId = ({ length = 4 }: { length?: number } = {}): string =>
  Math.floor(Math.random() * Math.pow(10, length)).toString();

/**
 * Generates a naming function and a random hash to help with naming collisions.
 * Notes about the session hash: Even with a unique baseName, the session hash
 * can be required when searching for objects created during the session. For
 * example, users can't be deleted and the users table is guaranteed to be dirty
 * when running tests more than once locally. The session hash helps avoid the
 * other dirty data without needing a specific baseName.
 * Here's an example:
 * `await userManagementPage.search.pwLocator.fill(usernamePrefix + sessionRandomHash);`
 * @returns A function that appends a timestamp and a short random hash to a base
 * name to avoid naming collisions, and a random hash to avoid searching for specific names.
 */
function genSafeName(): { sessionRandomHash: string; safeName: (baseName: string) => string } {
  // note 1: Include _ in the hash to avoid naming collisions. for example, we would want to avoid
  // searching for the name "admin", and _ helps with that.
  // note 2: This will not result in an extra call to Math.random per safeName call. It's a
  // one-time call per session.
  const sessionRandomHash = `_${randIdAlphanumeric()}`;
  /**
   * Appends a timestamp and a short random hash to a base name to avoid naming collisions.
   * @param baseName The base name to append the timestamp and hash to.
   * @returns The base name with appended timestamp and hash.
   */
  return {
    safeName: (baseName: string): string => {
      const timestamp = Date.now();
      return `${baseName}${sessionRandomHash}${randIdAlphanumeric()}_${timestamp}`;
    },
    sessionRandomHash,
  };
}

const { sessionRandomHash, safeName } = genSafeName();
export { sessionRandomHash, safeName };
