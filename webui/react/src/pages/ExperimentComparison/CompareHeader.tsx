import { Space } from 'antd';
import React from 'react';

import PageHeaderFoldable from 'components/PageHeaderFoldable';

import css from './CompareHeader.module.scss';

const ComparisonHeader: React.FC = () => {
  return (
    <>
      <PageHeaderFoldable

        leftContent={(
          <Space align="center" className={css.base}>

            <div className={css.id}>Experiment Comparison</div>
            <div className={css.name} />

          </Space>
        )}
      />
      {/* <ExperimentHeaderProgress experiment={experiment} /> */}
    </>
  );
};

export default ComparisonHeader;
