import * as io from 'io-ts';

import { applyMappers } from 'utils/data';

const ioAddress = io.type({
  address: io.string,
  city: io.string,
  state: io.string,
  zipcode: io.number,
});

type ioTypeAddress = io.TypeOf<typeof ioAddress>;

const data: ioTypeAddress = {
  address: '123 Sesame St',
  city: 'Manhattan',
  state: 'NY',
  zipcode: 10022,
};

const oneLineAddress = '123 Sesame St, Manhattan, NY 10022';

const addressToString = (address: ioTypeAddress): string => {
  return `${address.address}, ${address.city}, ${address.state} ${address.zipcode}`;
};

const stringToAddress = (str: string): ioTypeAddress | null => {
  const regex = /([\w\s]+), ([\w\s]+), ([A-Z]{2}) (\d{5})/i;
  const matches = str.match(regex);

  if (!matches || matches.length !== 5) return null;

  return {
    address: matches[1],
    city: matches[2],
    state: matches[3],
    zipcode: parseInt(matches[4]),
  };
};

const addressToCity = (address: ioTypeAddress): string => address.city;

describe('useRestApi', () => {
  describe('utility functions', () => {

    it('should modify data with one mapper', () => {
      const result = applyMappers(data, addressToString);
      expect(result).toBe(oneLineAddress);
    });

    it('should modify data with another mapper', () => {
      const result = applyMappers(oneLineAddress, stringToAddress);
      expect(result).toStrictEqual(data);
    });

    it('should modify data with multiple mapper', () => {
      const result = applyMappers(data, [ addressToString, stringToAddress, addressToCity ]);
      expect(result).toStrictEqual(data.city);
    });
  });
});
