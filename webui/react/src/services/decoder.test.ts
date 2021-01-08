import hparams from 'fixtures/hyperparameter-configs.json';
import experimentResps from 'fixtures/responses/experiment-details/set-a.json';
import * as ioTypes from 'ioTypes';

type FailReport<T = unknown> = {error: Error; sample: T;}

const tryOnSamples=
<T=unknown>(samples: T[], fn: (sample: T) => void): FailReport[] => {
  const fails: FailReport[] = [];
  samples.forEach(sample => {
    try {
      fn(sample);
    } catch(e) {
      fails.push({ error: e, sample });
    }
  });
  if (fails.length > 0) {
    const { sample, error } = fails[fails.length-1];
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
      ioTypes
        .decode<ioTypes.ioTypeHyperparameters>(ioTypes.ioHyperparameters, hparam);
    });
    expect(fails).toHaveLength(0);
  });

  it('Should decode experiment configs', () => {
    const fails = tryOnSamples(experimentResps.map(r => r.config), (config) => {
      ioTypes
        .decode<ioTypes.ioTypeExperimentConfig>(ioTypes.ioExperimentConfig, config);
    });
    expect(fails).toHaveLength(0);
  });
});
