import * as expect from 'expect';
import { waitFor, sleep } from './StepImplementation';

describe('waitFor', () => {
  let callCount = 0;
  const variableRequest = async () => {
    callCount++;
    await sleep(200);
    if (callCount !== 3) {
      throw new Error(`nope ${callCount}`);
    }
    return callCount;
  };

  beforeEach(() => {
    callCount = 0;
  });

  // it('should return with user error for sync requests', async () => {
  //   const req = () => {
  //     throw new Error('case 1');
  //   };

  //   expect(await waitFor(req, 500)).toThrowError('case 1');
  // });

  // it('should return with user error for async requests after timeout', async () => {
  //   const req = async () => {
  //     await sleep(200);
  //     throw new Error('case 2');
  //   };
  //   expect(await waitFor(req, 500)).toThrowError('case 2');
  // });

  // it('should timeout even if the first user request takes long', async () => {
  //   const req = async () => {
  //     await sleep(2000);
  //   };
  //   expect(await waitFor(req, 100)).toThrowError('timeout');
  // });

  it('should keep retrying user request', async () => {
    // expect(() => {
    //   throw new Error();
    // }).toThrow();
    expect(() => {
      waitFor(() => {
        throw new Error('e');
      }, 200);
    }).toThrow();
    // await waitFor(variableRequest, 500)).toThrow();
    // expect(callCount).toEqual(2);
  });

  // it('should stop retrying user request after success', async () => {
  //   await waitFor(variableRequest, 2000);
  //   expect(callCount).toEqual(3);
  // });
});
