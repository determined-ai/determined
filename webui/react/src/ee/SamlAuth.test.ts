import * as utils from './SamlAuth';

describe('SamlAuth', () => {
  describe('samlUrl', () => {
    const BASE_PATHS = ['/abc/def-ghi', '/HelloWorld/What%20is%20up?'];
    const QUERIES = [
      {
        default: 'columns=id&columns=user&sortDesc=false&tableLimit=20',
        encoded: 'columns%3Did%26columns%3Duser%26sortDesc%3Dfalse%26tableLimit%3D20',
      },
      {
        default: 'sortDesc=false&sortKey=SORT_BY_NAME&tags=mnist',
        encoded: 'sortDesc%3Dfalse%26sortKey%3DSORT_BY_NAME%26tags%3Dmnist',
      },
    ];

    it('should return base path only if no queries are provided', () => {
      for (const basePath of BASE_PATHS) {
        expect(utils.samlUrl(basePath)).toBe(basePath);
      }
    });

    it('should encode the query param', () => {
      const basePath = BASE_PATHS.first();
      for (const query of QUERIES) {
        const expected = `${basePath}?relayState=${query.encoded}`;
        expect(utils.samlUrl(basePath, query.default)).toBe(expected);
      }
    });
  });

  describe('handleRelayState', () => {
    const QUERIES_WITHOUT_RELAY = new URLSearchParams({ someKey: 'noRelayState' });
    const QUERIES_INPUT = new URLSearchParams({
      relayState: 'columns=id&columns=user&sortDesc=false&tableLimit=20',
      someKey: 'someValue',
    });
    const QUERIES_OUTPUT: { [key: string]: unknown } = {
      columns: ['id', 'user'],
      someKey: 'someValue',
      sortDesc: 'false',
      tableLimit: '20',
    };

    it('should return original queries object without relay state', () => {
      const result = utils.handleRelayState(QUERIES_WITHOUT_RELAY);
      for (const [key, val] of QUERIES_WITHOUT_RELAY.entries()) {
        expect(result.get(key)).toEqual(val);
      }
    });

    it('should decode and flatten relayState query param', () => {
      const result = utils.handleRelayState(QUERIES_INPUT);
      for (const key of result.keys()) {
        const val = result.getAll(key);
        expect(val.length === 1 ? val[0] : val).toStrictEqual(QUERIES_OUTPUT[key]);
      }
    });
  });
});
