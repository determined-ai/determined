import { render, screen } from '@testing-library/react';
import React from 'react';

import TimeDuration from 'components/TimeDuration';
import {
  DURATION_DAY,
  DURATION_HOUR,
  DURATION_MINUTE,
  DURATION_MONTH,
  DURATION_SECOND,
  DURATION_WEEK,
  DURATION_YEAR,
} from 'utils/datetime';

describe('TimeDuration', () => {
  it('should render a second', () => {
    render(<TimeDuration duration={DURATION_SECOND} />);
    expect(screen.getByText(/1s/i)).toBeInTheDocument();
  });

  it('should render a minute', () => {
    render(<TimeDuration duration={DURATION_MINUTE} />);
    expect(screen.getByText(/1m/i)).toBeInTheDocument();
  });

  it('should render a hour', () => {
    render(<TimeDuration duration={DURATION_HOUR} />);
    expect(screen.getByText(/1h/i)).toBeInTheDocument();
  });

  it('should render a day', () => {
    render(<TimeDuration duration={DURATION_DAY} />);
    expect(screen.getByText(/1d/i)).toBeInTheDocument();
  });

  it('should render a week', () => {
    render(<TimeDuration duration={DURATION_WEEK} />);
    expect(screen.getByText(/1w/i)).toBeInTheDocument();
  });

  it('should render a month', () => {
    render(<TimeDuration duration={DURATION_MONTH} />);
    expect(screen.getByText(/1mo/i)).toBeInTheDocument();
  });

  it('should render a year', () => {
    render(<TimeDuration duration={DURATION_YEAR} />);
    expect(screen.getByText(/1y/i)).toBeInTheDocument();
  });

  it('should render multiple units', () => {
    render(<TimeDuration duration={DURATION_YEAR + DURATION_MONTH} />);
    expect(screen.getByText(/1y 1mo/i)).toBeInTheDocument();
  });

  it('should render custom multiple units', () => {
    render(<TimeDuration duration={DURATION_YEAR + DURATION_MONTH + DURATION_WEEK} units={3} />);
    expect(screen.getByText(/1y 1mo 1w/i)).toBeInTheDocument();
  });
});
