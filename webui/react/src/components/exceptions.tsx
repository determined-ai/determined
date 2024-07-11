import Button from 'hew/Button';
import Message from 'hew/Message';
import React from 'react';

import Link from 'components/Link';
import useFeature from 'hooks/useFeature';
import { paths } from 'routes/utils';
import { capitalize } from 'utils/string';

export const NoExperiments: React.FC = () => {
  const f_flat_runs = useFeature().isOn('flat_runs');
  const entityCopy = f_flat_runs ? 'experiments' : 'searches';

  return (
    <Message
      action={
        <Link external path={paths.docs('/get-started/webui-qs.html')}>
          Quick Start Guide
        </Link>
      }
      description={`Keep track of ${entityCopy} you run in a project by connecting up your code.`}
      icon="experiment"
      title={`No ${capitalize(entityCopy)}`}
    />
  );
};

export const NoMatches: React.FC<{ clearFilters?: () => void }> = ({ clearFilters }) => (
  <Message
    action={<Button onClick={clearFilters}>Clear Filters</Button>}
    title="No Matching Results"
  />
);
export const Error: React.FC<{ fetchData?: () => void }> = ({ fetchData }) => (
  <Message
    action={<Button onClick={fetchData}>Retry</Button>}
    icon="error"
    title="Failed to Load Data"
  />
);
