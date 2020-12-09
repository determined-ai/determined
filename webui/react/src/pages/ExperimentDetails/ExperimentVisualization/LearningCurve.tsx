import React from 'react';

import LearningCurveChart from 'components/LearningCurveChart';
import Section from 'components/Section';

import css from './LearningCurve.module.scss';

const LearningCurve: React.FC = () => {
  return (
    <Section title="Learning Curve">
      <div className={css.base}>
        <LearningCurveChart />
      </div>
    </Section>
  );
};

export default LearningCurve;
