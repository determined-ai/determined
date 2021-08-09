import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { MINIMUM_PAGE_SIZE } from 'components/Table';
import useStorage from 'hooks/useStorage';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { parseUrl } from 'routes/utils';
import { ApiSorter } from 'services/types';
import { ExperimentBase, MetricName, MetricType, Pagination, TrialDetails } from 'types';
import { extractMetricNames } from 'utils/trial';

import TrialChart from './TrialChart';
import TrialDetailsWorkloads from './TrialDetailsWorkloads';

const defaultSorter: ApiSorter = {
  descend: true,
  key: 'batches',
};

export enum TrialInfoFilter {
  Checkpoint = 'Has Checkpoint',
  Validation = 'Has Validation',
  CheckpointOrValidation = 'Has Checkpoint or Validation',
}

const URL_ALL = 'all';

const STORAGE_CHART_METRICS_KEY = 'metrics/chart';
const STORAGE_PATH = 'trial-detail';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_CHECKPOINT_VALIDATION_KEY = 'checkpoint-validation';
const STORAGE_SORTER_KEY = 'sorter';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialDetailsOverview: React.FC<Props> = ({ experiment, trial }: Props) => {
  const [ defaultMetrics, setDefaultMetrics ] = useState<MetricName[]>([]);
  const storageMetricsPath = experiment ? `experiments/${experiment.id}` : undefined;
  const storage = useStorage(STORAGE_PATH);
  const [ isUrlParsed, setIsUrlParsed ] = useState(false);

  const storageChartMetricsKey =
    storageMetricsPath && `${storageMetricsPath}/${STORAGE_CHART_METRICS_KEY}`;

  const validationMetric = useMemo(
    () => experiment?.config?.searcher.metric,
    [ experiment?.config?.searcher.metric ],
  );
  const metricNames = useMemo(() => extractMetricNames(
    trial?.workloads || [],
  ), [ trial?.workloads ]);
  const defaultMetric = useMemo(() => {
    return metricNames.find(metricName => (
      metricName.name === validationMetric && metricName.type === MetricType.Validation
    ));
  }, [ metricNames, validationMetric ]);
  const fallbackMetric = metricNames && metricNames.length !== 0 ? metricNames[0] : undefined;
  const initMetric = defaultMetric || fallbackMetric;
  const [ metrics, setMetrics ] = useState<MetricName[]>(
    storage.getWithDefault(storageChartMetricsKey || '', initMetric ? [ initMetric ] : []),
  );
  const initFilter = storage.getWithDefault(
    STORAGE_CHECKPOINT_VALIDATION_KEY,
    TrialInfoFilter.CheckpointOrValidation,
  );
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ sorter, setSorter ] = useState<ApiSorter>(initSorter);
  const [ pagination, setPagination ] = useState<Pagination>(
    { limit: initLimit, offset: 0 },
  );
  const [ showFilter, setShowFilter ] = useState(initFilter);

  /*
   * When filters changes update the page URL.
   */
  useEffect(() => {
    if (!isUrlParsed) return;

    const searchParams = new URLSearchParams;
    const url = parseUrl(window.location.href);

    // limit
    searchParams.append('limit', pagination.limit.toString());

    // offset
    searchParams.append('offset', pagination.offset.toString());

    // sortDesc
    searchParams.append('sortDesc', sorter.descend ? '1' : '0');

    // sortKey
    searchParams.append('sortKey', sorter.key || '');

    // filters
    searchParams.append('filter', showFilter ? showFilter : URL_ALL);

    // metrics
    if (metrics && metrics.length > 0) {
      metrics.forEach(metric => searchParams.append('metric', metric.name + '|' + metric.type));
    } else {
      searchParams.append('state', URL_ALL);
    }

    window.history.pushState(
      {},
      '',
      url.origin + url.pathname + '?' + searchParams.toString(),
    );
  }, [ metrics, isUrlParsed, pagination, showFilter, sorter ]);

  /*
     * On first load: if filters are specified in URL, override default.
     */
  useEffect(() => {
    if (isUrlParsed) return;

    // If search params are not set, we default to user preferences
    const url = parseUrl(window.location.href);
    if (url.search === '') {
      setIsUrlParsed(true);
      return;
    }

    const urlSearchParams = url.searchParams;

    // limit
    const limit = urlSearchParams.get('limit');
    if (limit != null && !isNaN(parseInt(limit))) {
      pagination.limit = parseInt(limit);
    }

    // offset
    const offset = urlSearchParams.get('offset');
    if (offset != null && !isNaN(parseInt(offset))) {
      pagination.offset = parseInt(offset);
    }

    // sortDesc
    const sortDesc = urlSearchParams.get('sortDesc');
    if (sortDesc != null) {
      sorter.descend = (sortDesc === '1');
    }

    // filter
    const filter = urlSearchParams.get('filter');
    if (filter != null) {
      setShowFilter(filter as TrialInfoFilter);
    }

    // metrics
    const visibleMetrics = urlSearchParams.getAll('metric');
    let metrics = defaultMetrics;
    if (visibleMetrics != null) {
      metrics = (visibleMetrics.map(metric => {
        const splitMetric = metric.split('|');
        return { name: splitMetric[0], type: splitMetric[1] as MetricType };
      }));
    }

    // sortKey
    const sortKey = urlSearchParams.get('sortKey');
    if (sortKey != null &&
      [ 'batches', 'state', ...metrics.map(metric => metric.name) ].includes(sortKey)) {
      sorter.key = sortKey;
    }

    setIsUrlParsed(true);
    setPagination(pagination);
    setSorter(sorter);
    setMetrics(metrics);
  }, [ defaultMetrics, isUrlParsed, pagination, showFilter, sorter, metrics ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    setMetrics(value);

    if (storageChartMetricsKey) storage.set(storageChartMetricsKey, value);
  }, [ storage, storageChartMetricsKey ]);

  // Default to selecting config search metric only.
  useEffect(() => {
    const searcherName = experiment.config?.searcher?.metric;
    const defaultMetric = metricNames.find(metricName => {
      return metricName.name === searcherName && metricName.type === MetricType.Validation;
    });
    const defaultMetrics = defaultMetric ? [ defaultMetric ] : [];
    setDefaultMetrics(defaultMetrics);
  }, [ experiment, metricNames, storage ]);

  return (
    <>
      <TrialInfoBox experiment={experiment} trial={trial} />
      <TrialChart
        defaultMetricNames={defaultMetrics}
        handleMetricChange={handleMetricChange}
        metricNames={metricNames}
        metrics={metrics}
        workloads={trial?.workloads} />
      <TrialDetailsWorkloads
        defaultMetrics={defaultMetrics}
        experiment={experiment}
        handleMetricChange={handleMetricChange}
        metricNames={metricNames}
        metrics={metrics}
        pagination={pagination}
        setPagination={setPagination}
        setShowFilter={setShowFilter}
        setSorter={setSorter}
        showFilter={showFilter}
        sorter={sorter}
        trial={trial} />
    </>
  );
};

export default TrialDetailsOverview;
