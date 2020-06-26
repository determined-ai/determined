import React from 'react';
import { useParams } from 'react-router';

import Page from 'components/Page';

interface Params {
  experimentId: string;
}

const ExperimentDetails: React.FC = () => {
  const { experimentId } = useParams<Params>();
  return (
    <Page title={`Experiment ${experimentId}`} />
  );
};

export default ExperimentDetails;
