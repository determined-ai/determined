import { Button, Col, Row, Tooltip } from 'antd';
import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import Link from 'components/Link';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import {
  defaultRowClassName, getFullPaginationConfig, MINIMUM_PAGE_SIZE,
} from 'components/Table';
import { Renderer } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import ExperimentChart from 'pages/ExperimentDetails/ExperimentChart';
import ExperimentInfoBox from 'pages/ExperimentDetails/ExperimentInfoBox';
import { paths } from 'routes/utils';
import { getExpTrials } from 'services/api';
import { V1GetExperimentTrialsRequestSortBy } from 'services/api-ts-sdk';
import { ApiSorter } from 'services/types';
import { validateDetApiEnum } from 'services/utils';
import {
  CheckpointWorkloadExtended, ExperimentBase, Pagination, TrialItem, ValidationHistory,
} from 'types';
import { getMetricValue, terminalRunStates } from 'utils/types';

import { columns as defaultColumns } from './ExperimentOverview.table';

interface Props {
  experiment: ExperimentBase;
  onTagsChange: () => void;
  validationHistory: ValidationHistory[];
}

const STORAGE_PATH = 'experiment-detail';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';

const defaultSorter: ApiSorter<V1GetExperimentTrialsRequestSortBy> = {
  descend: true,
  key: V1GetExperimentTrialsRequestSortBy.ID,
};

const ExperimentOverview: React.FC<Props> = ({
  experiment,
  validationHistory,
  onTagsChange,
}: Props) => {
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ pagination, setPagination ] = useState<Pagination>({ limit: initLimit, offset: 0 });
  const [ total, setTotal ] = useState(0);
  const [ sorter, setSorter ] = useState(initSorter);
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointWorkloadExtended>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ trials, setTrials ] = useState<TrialItem[]>();
  const [ canceler ] = useState(new AbortController());

  const columns = useMemo(() => {
    const { metric } = experiment.config?.searcher || {};

    const idRenderer: Renderer<TrialItem> = (_, record) => (
      <Link path={paths.trialDetails(record.id, experiment.id)}>
        <span>Trial {record.id}</span>
      </Link>
    );

    const validationRenderer = (key: string) => {
      return function renderer (_: string, record: TrialItem): React.ReactNode {
        /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
        const value = getMetricValue((record as any)[key], metric);
        return value && <HumanReadableFloat num={value} />;
      };
    };

    const checkpointRenderer = (_: string, record: TrialItem): React.ReactNode => {
      if (!record.bestAvailableCheckpoint) return;
      const checkpoint: CheckpointWorkloadExtended = {
        ...record.bestAvailableCheckpoint,
        experimentId: experiment.id,
        trialId: record.id,
      };
      return (
        <Tooltip title="View Checkpoint">
          <Button
            aria-label="View Checkpoint"
            icon={<Icon name="checkpoint" />}
            onClick={e => handleCheckpointShow(e, checkpoint)} />
        </Tooltip>
      );
    };

    const newColumns = [ ...defaultColumns ].map(column => {
      column.sortOrder = null;
      if (column.key === 'checkpoint') {
        column.render = checkpointRenderer;
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.ID) {
        column.render = idRenderer;
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.BESTVALIDATIONMETRIC) {
        column.render = validationRenderer('bestValidationMetric');
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.LATESTVALIDATIONMETRIC) {
        column.render = validationRenderer('latestValidationMetric');
      }
      if (column.key === sorter.key) {
        column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      }
      return column;
    });

    return newColumns;
  }, [ experiment.config, experiment.id, sorter ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, sorter) => {
    if (Array.isArray(sorter)) return;

    const { columnKey, order } = sorter as SorterResult<TrialItem>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    storage.set(STORAGE_SORTER_KEY, { descend: order === 'descend', key: columnKey as string });
    setSorter({
      descend: order === 'descend',
      key: columnKey as V1GetExperimentTrialsRequestSortBy,
    });

    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPagination(prev => ({
      ...prev,
      limit: tablePagination.pageSize,
      offset: (tablePagination.current - 1) * tablePagination.pageSize,
    }));
  }, [ columns, setSorter, storage ]);

  const handleCheckpointShow = (
    event: React.MouseEvent,
    checkpoint: CheckpointWorkloadExtended,
  ) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };

  const handleCheckpointDismiss = useCallback(() => setShowCheckpoint(false), []);

  const fetchExperimentTrials = useCallback(async () => {
    try {
      const { trials: experimentTrials, pagination: responsePagination } = await getExpTrials(
        {
          id: experiment.id,
          limit: pagination.limit,
          offset: pagination.offset,
          orderBy: sorter.descend ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetExperimentTrialsRequestSortBy, sorter.key),
        },
        { signal: canceler.signal },
      );
      setTotal(responsePagination?.total || 0);
      setTrials(experimentTrials);
      setIsLoading(false);
    } catch (e) {
      handleError({
        message: `Unable to fetch experiments ${experiment.id} trials.`,
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ experiment.id, canceler, pagination, sorter ]);

  const { stopPolling } = usePolling(fetchExperimentTrials);

  // Get new trials based on changes to the pagination and sorter.
  useEffect(() => {
    fetchExperimentTrials();
    setIsLoading(true);
  }, [ fetchExperimentTrials ]);

  useEffect(() => {
    if (terminalRunStates.has(experiment.state)) stopPolling({ terminateGracefully: true });
  }, [ experiment.state, stopPolling ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <>
      <Row gutter={[ 16, 16 ]}>
        <Col lg={10} span={24} xl={8} xxl={6}>
          <ExperimentInfoBox
            experiment={experiment}
            onTagsChange={onTagsChange}
          />
        </Col>
        <Col lg={14} span={24} xl={16} xxl={18}>
          <ExperimentChart
            startTime={experiment.startTime}
            validationHistory={validationHistory}
            validationMetric={experiment.config?.searcher.metric} />
        </Col>
        <Col span={24}>
          <Section title="Trials">
            <ResponsiveTable
              columns={columns}
              dataSource={trials}
              loading={isLoading}
              pagination={getFullPaginationConfig(pagination, total)}
              rowClassName={defaultRowClassName({ clickable: false })}
              rowKey="id"
              showSorterTooltip={false}
              size="small"
              onChange={handleTableChange} />
          </Section>
        </Col>
      </Row>
      {activeCheckpoint && <CheckpointModal
        checkpoint={activeCheckpoint}
        config={experiment.config}
        show={showCheckpoint}
        title={`Best Checkpoint for Trial ${activeCheckpoint.trialId}`}
        onHide={handleCheckpointDismiss} />}
    </>
  );
};

export default ExperimentOverview;
