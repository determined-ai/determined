// @vitest-environment node
// we're using node because the jsdom version of the abortcontroller doesn't have reasons on abortsignals
import fc from 'fast-check';
import { zip } from 'lodash';

import { mergeAbortControllers } from 'utils/mergeAbortControllers';

// arbitrary to generate a list of abort controllers to pass to mergeAbortControllers
const argArb = fc.uniqueArray(
  fc.constant(() => new AbortController()).map((f) => f()),
  { minLength: 1 },
);

// return a subset of the above to control
const argArbWithSelection = (n?: number) =>
  argArb.chain((arr) =>
    fc.tuple(fc.constant(arr), fc.shuffledSubarray(arr, { maxLength: n, minLength: 1 })),
  );

// the above, but the subset from the above returns with unique reason values to
// verify which abortController was the first to abort
const argArbWithSelectionAndReasons = (n?: number) =>
  argArbWithSelection(n).chain(([args, selection]) => {
    const reasonsArb = fc.uniqueArray(fc.anything(), {
      maxLength: selection.length,
      minLength: selection.length,
    });
    const selectionAndReasonArb = reasonsArb
      .map((reasons) => zip(selection, reasons))
      .filter((tups): tups is [AbortController, unknown][] =>
        tups.every((tup) => tup.every((c) => c !== undefined)),
      );

    return fc.tuple(fc.constant(args), selectionAndReasonArb);
  });

describe('mergeAbortControllers', () => {
  it('merged abort controller aborts if any constituent aborts', () => {
    fc.assert(
      fc.property(argArbWithSelection(1), ([args, abortControllers]) => {
        const [abortController] = abortControllers;
        const result = mergeAbortControllers(...args);

        abortController.abort();
        expect(result.signal.aborted).toBe(true);
      }),
    );
  });
  it('merged abort controller aborts with constituent reason', () => {
    fc.assert(
      fc.property(argArbWithSelectionAndReasons(1), ([args, abortControllers]) => {
        const [[abortController, reason]] = abortControllers;
        const result = mergeAbortControllers(...args);

        abortController.abort(reason);
        expect(abortController.signal.reason).toBe(reason);
        expect(result.signal.reason).toBe(abortController.signal.reason);
      }),
    );
  });
  it('merged abort controller only reflects the first abort', () => {
    fc.assert(
      fc.property(argArbWithSelectionAndReasons(), ([args, abortControllers]) => {
        const [[firstAbortController]] = abortControllers;
        const result = mergeAbortControllers(...args);

        abortControllers.forEach(([abortController, reason]) => {
          abortController.abort(reason);
        });
        expect(result.signal.reason).toBe(firstAbortController.signal.reason);
      }),
    );
  });

  it('merging an aborted controller results in an aborted controller', () => {
    fc.assert(
      fc.property(argArbWithSelection(1), ([args, abortControllers]) => {
        const [abortController] = abortControllers;
        abortController.abort();
        const result = mergeAbortControllers(...args);

        expect(result.signal.aborted).toBe(true);
        expect(result.signal.reason).toBe(abortController.signal.reason);
      }),
    );
  });
});
