import Badge from 'hew/Badge';
import Icon from 'hew/Icon';
import React from 'react';
import { Link } from 'react-router-dom';

import ExperimentIcons from 'components/ExperimentIcons';
import useFeature from 'hooks/useFeature';
import { paths } from 'routes/utils';
import { ExperimentBase, TrialDetails } from 'types';
import { hex2hsl } from 'utils/color';

import css from './TrialHeaderLeft.module.scss';

interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialHeaderLeft: React.FC<Props> = ({ experiment, trial }: Props) => {
  const f_flat_runs = useFeature().isOn('flat_runs');

  return (
    <div className={css.base}>
      <Link className={css.experiment} to={paths.experimentDetails(trial.experimentId)}>
        {f_flat_runs ? 'Search' : 'Experiment'} {trial.experimentId} | {experiment.name}
      </Link>
      <Icon decorative name="arrow-right" size="tiny" />
      <div className={css.trial}>
        <ExperimentIcons state={trial.state} />
        <div>Trial {trial.id}</div>
        {trial.logSignals?.map((s) => (
          <Badge backgroundColor={hex2hsl('#CC0000')} key={s} text={s} />
        ))}
      </div>
    </div>
  );
};

export default TrialHeaderLeft;
