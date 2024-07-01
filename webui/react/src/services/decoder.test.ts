import hparams from 'fixtures/hyperparameter-configs.json';
import experimentResps from 'fixtures/responses/experiment-details/set-a.json';
import * as ioTypes from 'ioTypes';
import { V1ExperimentActionResult, V1RunActionResult } from 'services/api-ts-sdk';
import * as decoder from 'services/decoder';

type FailReport<T = unknown> = { error: Error; sample: T };

const tryOnSamples = <T = unknown>(samples: T[], fn: (sample: T) => void): FailReport[] => {
  const fails: FailReport[] = [];
  samples.forEach((sample) => {
    try {
      fn(sample);
    } catch (e) {
      fails.push({ error: e as Error, sample });
    }
  });
  if (fails.length > 0) {
    const { sample, error } = fails.last();
    /* eslint-disable no-console */
    console.error(error);
    console.log('Sample:', sample);
    /* eslint-enable no-console */
  }
  return fails;
};

describe('Decoder', () => {
  it('Should decode seeded hyperparameters', () => {
    const fails = tryOnSamples(hparams, (hparam) => {
      ioTypes.decode<ioTypes.ioTypeHyperparameters>(ioTypes.ioHyperparameters, hparam);
    });
    expect(fails).toHaveLength(0);
  });

  it('Should decode experiment configs', () => {
    const fails = tryOnSamples(
      experimentResps.map((r) => r.config),
      (config) => {
        ioTypes.decode<ioTypes.ioTypeExperimentConfig>(ioTypes.ioExperimentConfig, config);
      },
    );
    expect(fails).toHaveLength(0);
  });

  describe('mapV1ActionResults', () => {
    it('should work with Sdk.V1ExperimentActionResult[] input', () => {
      const result: V1ExperimentActionResult[] = [
        { error: '', id: 1 },
        { error: '', id: 2 },
        { error: 'error', id: 3 },
      ];

      const expected = decoder.mapV1ActionResults(result);
      expect(expected).toStrictEqual({
        failed: [{ error: 'error', id: 3 }],
        successful: [1, 2],
      });
    });

    it('should work with Sdk.V1RunActionResult[] input', () => {
      const result: V1RunActionResult[] = [
        { error: '', id: 1 },
        { error: '', id: 2 },
        { error: 'error', id: 3 },
      ];

      const expected = decoder.mapV1ActionResults(result);
      expect(expected).toStrictEqual({
        failed: [{ error: 'error', id: 3 }],
        successful: [1, 2],
      });
    });

    it('should work with empty input', () => {
      const expected = decoder.mapV1ActionResults([]);
      expect(expected).toStrictEqual({
        failed: [],
        successful: [],
      });
    });

    it('should work with all successful input', () => {
      const result: V1RunActionResult[] = [
        { error: '', id: 1 },
        { error: '', id: 2 },
        { error: '', id: 3 },
        { error: '', id: 4 },
        { error: '', id: 5 },
      ];

      const expected = decoder.mapV1ActionResults(result);
      expect(expected).toStrictEqual({
        failed: [],
        successful: [1, 2, 3, 4, 5],
      });
    });

    it('should work with all failed input', () => {
      const result: V1RunActionResult[] = [
        { error: 'oh no', id: 1 },
        { error: 'yare yare', id: 2 },
        { error: 'error', id: 3 },
        { error: 'a', id: 4 },
        { error: 'エラー', id: 5 },
      ];

      const expected = decoder.mapV1ActionResults(result);
      expect(expected).toStrictEqual({
        failed: [
          { error: 'oh no', id: 1 },
          { error: 'yare yare', id: 2 },
          { error: 'error', id: 3 },
          { error: 'a', id: 4 },
          { error: 'エラー', id: 5 },
        ],
        successful: [],
      });
    });
  });
});
