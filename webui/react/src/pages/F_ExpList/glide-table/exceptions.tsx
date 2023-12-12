import Button from 'hew/Button';
import Message from 'hew/Message';
import React from 'react';

import Link from 'components/Link';
import { paths } from 'routes/utils';

export const NoExperiments: React.FC = () => {
  return (
    <Message
      action={
        <Link external path={paths.docs('/tutorials/quickstart-mdldev.html')}>
          Quick Start Guide
        </Link>
      }
      description="Keep track of experiments you run in a project by connecting up your code."
      icon="experiment"
      title="No Experiments"
    />
  );
};

export const NoMatches: React.FC<{ clearFilters?: () => void }> = ({ clearFilters }) => (
  <Message
    action={<Button onClick={clearFilters}>Clear Filters</Button>}
    title="No Matching Results"
  />
);
export const Error: React.FC<{ fetchExperiments?: () => void }> = ({ fetchExperiments }) => (
  <Message
    action={<Button onClick={fetchExperiments}>Retry</Button>}
    icon="error"
    title="Failed to Load Data"
  />
);
