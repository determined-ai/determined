import React from 'react';

import ExperimentsTable from 'components/ExperimentTable';
import Page from 'components/Page';
import Experiments from 'contexts/Experiments';

const ExperimentList: React.FC = () => {
  const experiments = Experiments.useStateContext();
  return (
    <Page title="Experiments">
      {<ExperimentsTable experiments={experiments.data} />}
    </Page>
  );
};

export default ExperimentList;
