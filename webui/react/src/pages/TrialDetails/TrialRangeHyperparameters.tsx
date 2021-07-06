import { Tooltip } from 'antd';
import React, { useMemo } from 'react';

import HumanReadableFloat from 'components/HumanReadableFloat';
import Section from 'components/Section';
import {
  ExperimentBase, ExperimentHyperParamType, TrialDetails,
} from 'types';
import { clamp } from 'utils/number';

import css from './TrialRangeHyperparameters.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

interface HyperParameter {
  name: string;
  range: [number, number];
  type: ExperimentHyperParamType;
  val: string;
  vals: string[];
}

const TrialRangeHyperparameters: React.FC<Props> = ({ experiment, trial }: Props) => {
  const hyperparameters: HyperParameter[] = useMemo(() => {
    const config = Object.entries(experiment.config.hyperparameters).map(([ name, value ]) => {
      return { name, value };
    });
    const value = Object.entries(trial.hparams).map(([ name, value ]) => {
      return { name, value };
    });
    return config.map(hp => {
      return (
        {
          name: hp.name,
          range: [ hp.value.minval || 0.00001, hp.value.maxval || 1 ],
          type: (hp.value.type === ExperimentHyperParamType.Log &&
            [ hp.value.minval|| 0.00001, hp.value.maxval || 1 ].some(val => val < 0))
            ? ExperimentHyperParamType.Double : hp.value.type,
          val: String(value.find(ob => ob.name === hp.name)?.value || 0),
          vals: hp.value.vals?.map(val => String(val)) ||
          [ String(hp.value.minval || 0), String(hp.value.maxval || 1) ],
        }
      );
    });
  }, [ experiment.config.hyperparameters, trial.hparams ]);

  return (
    <Section bodyBorder bodyScroll>
      <div className={css.container}>
        {hyperparameters.map(hp => <div key={hp.name}>
          <HyperparameterRange hp={hp} />
        </div>)}
      </div>
    </Section>
  );
};

interface RangeProps {
  hp: HyperParameter
}

const HyperparameterRange:React.FC<RangeProps> = ({ hp }: RangeProps) => {
  const pointerPosition = useMemo(() => {
    switch (hp.type) {
      case ExperimentHyperParamType.Constant:
        return 0.5;
      case ExperimentHyperParamType.Categorical:
      {
        const idx = hp.vals.indexOf(hp.val);
        return ((idx === -1 ? 0 : idx)/(hp.vals.length-1));
      }
      case ExperimentHyperParamType.Log:
        return clamp(1-Math.log(parseFloat(hp.val)/hp.range[0])/
            (Math.log(hp.range[1]/hp.range[0])), 0, 1);
      default:
        return clamp(1-(parseFloat(hp.val)-hp.range[0])/
            (hp.range[1] - hp.range[0]), 0, 1);
    }
  }, [ hp ]);

  return (
    <div className={css.hpContainer}>
      <p className={css.title}>{hp.name}</p>
      <div className={css.innerContainer}>
        <ValuesTrack hp={hp} />
        <MainTrack hp={hp} />
        <div className={css.pointerTrack}>
          <Pointer
            containerStyle={{ transform: `translateY(${270*pointerPosition}px)` }}
            content={<ParsedHumanReadableValue hp={hp} />} />
        </div>
      </div>
    </div>
  );
};

interface TrackProps {
  hp: HyperParameter
}

const ValuesTrack: React.FC<TrackProps> = ({ hp }: TrackProps) => {
  switch(hp.type) {
    case ExperimentHyperParamType.Constant:
      return <div className={css.valuesTrack} />;
    case ExperimentHyperParamType.Categorical:
      return <div className={css.valuesTrack}>
        {hp.vals.map(option =>
          <p className={css.text} key={option.toString()}>{option}</p>)}
      </div>;
    case ExperimentHyperParamType.Log:
      return <div className={css.valuesTrack}>
        {(new Array(Math.floor(Math.log10((hp.range[1])/(hp.range[0]))+1))).fill(null)
          .map((_, idx) =>
            <p className={css.text} key={idx}>
              {String((hp.range[1])/(10**idx)).length > 4 ?
                ((hp.range[1])/(10**idx)).toExponential() :
                (hp.range[1])/(10**idx)}
            </p>)}
      </div>;
    default:
      return <div className={css.valuesTrack}>
        <p className={css.text}>{hp.range[1]}</p>
        <p className={css.text}>{hp.range[0]}</p>
      </div>;
  }
};

const MainTrack: React.FC<TrackProps> = ({ hp }: TrackProps) => {
  let trackType;
  let content;
  switch(hp.type) {
    case ExperimentHyperParamType.Categorical:
      trackType = css.grayTrack;
      content = hp.vals.map(option =>
        <div
          className={css.trackOption}
          key={option.toString()}
        />);
      break;
    case ExperimentHyperParamType.Constant:
      trackType = css.constantTrack;
      content = <div
        className={css.trackOption}
      />;
      break;
    case ExperimentHyperParamType.Log:
      trackType = css.blueTrack;
      content = (new Array(Math.floor(Math.log10((hp.range[1])/(hp.range[0]))+1)))
        .fill(null)
        .map((_, idx) =>
          <div className={css.tick} key={idx} />);
      break;
    default:
      trackType = css.blueTrack;
      content = hp.vals.map(option =>
        <div
          className={css.trackOption}
          key={option.toString()}
        />);
  }
  return (
    <div className={trackType}>
      {content}
    </div>);
};

interface PHRVProps {
  hp: HyperParameter
}

const ParsedHumanReadableValue: React.FC<PHRVProps> = ({ hp }: PHRVProps) => {
  switch (hp.type) {
    case ExperimentHyperParamType.Categorical:
      return <p className={css.text}>{hp.val}</p>;
    case ExperimentHyperParamType.Constant:
      return <p className={css.text}>{hp.val}</p>;
    case ExperimentHyperParamType.Double:
      return (
        <p className={css.text}>
          <HumanReadableFloat num={parseFloat(hp.val as string)} precision={3} />
        </p>
      );
    case ExperimentHyperParamType.Int:
      return <p className={css.text}>{parseInt(hp.val as string)}</p>;
    case ExperimentHyperParamType.Log:
      return (
        <Tooltip title={hp.val}>
          <p className={css.text}>{parseFloat(hp.val as string).toExponential(2)}</p>
        </Tooltip>
      );
    default:
      return <p className={css.text}>{hp.val}</p>;
  }
};

interface PointerProps {
  containerStyle: React.CSSProperties;
  content: JSX.Element;
}

const Pointer: React.FC<PointerProps> = ({ containerStyle, content }: PointerProps) => {
  return (
    <div className={css.pointerContainer} style={containerStyle}>
      <div className={css.pointerArrow} />
      <div className={css.pointerText}>
        {content}
      </div>
    </div>
  );
};

export default TrialRangeHyperparameters;
