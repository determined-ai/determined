import React from 'react';
import { Link } from 'react-router-dom';

import ExperimentIcons from 'components/ExperimentIcons';
import Icon from 'components/kit/Icon';
import css from 'pages/TrialDetails/Header/TrialHeaderLeft.module.scss';
import { paths } from 'routes/utils';
import { ExperimentBase, TrialDetails } from 'types';

interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialHeaderLeft: React.FC<Props> = ({ experiment, trial }: Props) => {
  return (
    <div className={css.base}>
      <Link className={css.experiment} to={paths.experimentDetails(trial.experimentId)}>
        Experiment {trial.experimentId} | {experiment.name}
      </Link>
      <Icon decorative name="arrow-right" size="tiny" />
      <div className={css.trial}>
        <ExperimentIcons state={trial.state} />
        <div>Trial {trial.id}</div>
      </div>
    </div>
  );
};

export default TrialHeaderLeft;
