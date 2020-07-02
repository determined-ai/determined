import { Table } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import { makeClickHandler } from 'components/Link';
import linkCss from 'components/Link.module.scss';
import Page from 'components/Page';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { getExperimentSummaries } from 'services/api';
import { ExperimentsParams } from 'services/types';
import { Experiment, ExperimentItem } from 'types';
import { processExperiments } from 'utils/task';

import { columns } from './ExperimentList.table';

const ExperimentList: React.FC = () => {
  const users = Users.useStateContext();
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>([]);
  const [ experimentsResponse, requestExperiments ] =
    useRestApiSimple<ExperimentsParams, Experiment[]>(getExperimentSummaries, {});

  const fetchExperiments = useCallback((): void => {
    requestExperiments({});
  }, [ requestExperiments ]);

  usePolling(fetchExperiments);

  useEffect(() => {
    const experiments = processExperiments(experimentsResponse.data || [], users.data || []);
    setExperiments(experiments);
  }, [ experimentsResponse, setExperiments, users ]);

  const handleTableRow = useCallback((record: ExperimentItem) => ({
    onClick: makeClickHandler(record.url as string),
  }), []);

  return (
    <Page title="Experiments">
      <Table
        columns={columns}
        dataSource={experiments}
        loading={experiments === undefined}
        rowClassName={(): string => linkCss.base}
        rowKey="id"
        size="small"
        onRow={handleTableRow} />
    </Page>
  );
};

export default ExperimentList;
