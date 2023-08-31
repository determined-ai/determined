import { TrialHParams } from 'pages/ExperimentDetails/ExperimentVisualization/HpTrialTable';
import { getCustomSearchVaryingHPs } from 'pages/ExperimentDetails/ExperimentVisualization/LearningCurve';

describe('Custom Search HPs filter out constants across trials', () => {
  it("doesn't show hparams that are the same across trials", () => {
    const test: TrialHParams[] = [
      {
        hparams: {
          const_int: 3,
          uniq_int: 3,
        },
        id: 0,
        metric: 123,
      },
      {
        hparams: {
          const_int: 3,
          uniq_int: 4,
        },
        id: 0,
        metric: 123,
      },
    ];
    const ret: string[] = Object.keys(getCustomSearchVaryingHPs(test));
    ret.sort();
    const expected: string[] = ['uniq_int'];
    expected.sort();
    expect(ret).toStrictEqual(expected);
  });

  it('shows all primitive params if there is only one result so far', () => {
    const test: TrialHParams[] = [
      {
        hparams: {
          const_int: 3,
          uniq_int: 3,
        },
        id: 0,
        metric: 123,
      },
    ];
    const ret: string[] = Object.keys(getCustomSearchVaryingHPs(test));
    ret.sort();
    const expected: string[] = ['const_int', 'uniq_int'];
    expected.sort();
    expect(ret).toStrictEqual(expected);
  });

  it('can handle unique floats, strings, lists and bools', () => {
    const test: TrialHParams[] = [
      {
        hparams: {
          const_bool: true,
          const_float: 3.5,
          const_list: [1, 2],
          const_str: 'test',
          uniq_bool: true,
          uniq_float: 3.5,
          uniq_list: [1, 2],
          uniq_str: 'hello',
        },
        id: 0,
        metric: 123,
      },
      {
        hparams: {
          const_bool: true,
          const_float: 3.5,
          const_list: [1, 2],
          const_str: 'test',
          uniq_bool: false,
          uniq_float: 4.0,
          uniq_list: [1, 3],
          uniq_str: 'goodbye',
        },
        id: 0,
        metric: 123,
      },
    ];
    const ret: string[] = Object.keys(getCustomSearchVaryingHPs(test));
    ret.sort();
    const expected: string[] = ['uniq_float', 'uniq_str', 'uniq_bool', 'uniq_list'];
    expected.sort();
    expect(ret).toStrictEqual(expected);
  });

  it('ignores dictionaries that are not flattened', () => {
    const test: TrialHParams[] = [
      {
        hparams: {
          'dict': {
            a: 'b',
          },
          'dict.a': 'b',
        },
        id: 0,
        metric: 123,
      },
    ];
    const ret: string[] = Object.keys(getCustomSearchVaryingHPs(test));
    ret.sort();
    const expected: string[] = ['dict.a'];
    expected.sort();
    expect(ret).toStrictEqual(expected);
  });
});
