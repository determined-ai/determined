/* eslint-disable @typescript-eslint/no-explicit-any */
enum V1OrderBy {
  UNSPECIFIED = <any> 'ORDER_BY_UNSPECIFIED',
  ASC = <any> 'ORDER_BY_ASC',
  DESC = <any> 'ORDER_BY_DESC'
}

enum ProtobufNullValue {
  NULLVALUE = <any> 'NULL_VALUE'
}

import { DetError } from './error';
import * as service from './service';

describe('Service Utilities', () => {

  describe('isAuthFailure', () => {
    it('marks false for 200s', () => {

      const response = new Response('ok', { status: 200 });

      expect(service.isAuthFailure(response)).toBe(false);
    });

    it('marks false for 404s', () => {

      const response = new Response('not found', { status: 404 });

      expect(service.isAuthFailure(response)).toBe(false);
    });
    it('marks true for 401s', () => {

      const response = new Response('unauthorized', { status: 401 });

      expect(service.isAuthFailure(response)).toBe(true);
    });
    it('marks true for external 500s', () => {

      const response = new Response('external request failure', { status: 500 });

      expect(service.isAuthFailure(response, true)).toBe(true);
    });
  });

  describe('isApiResponse', () => {
    it('returns true for response', () => {
      const response = new Response('response text', { status: 200 });
      expect(service.isApiResponse(response)).toBe(true);
    });

    it('returns false for error', () => {
      const notResponse = new Error('Bad Error, go home');
      expect(service.isApiResponse(notResponse)).toBe(false);
    });
  });

  describe('isNotFound ', () => {
    it('response true', () => {
      const response = new Response('unauthorized', { status: 404 });
      expect(service.isNotFound(response)).toBe(true);
    });

    it('response false', () => {
      const response = new Response('unauthorized', { status: 400 });
      expect(service.isNotFound(response)).toBe(false);
    });

    it('returns true for det errors with not found in message', () => {
      const e = new DetError(
        'could not do',
        { publicMessage: 'could not do because Not Found', silent: true },
      );
      expect(service.isNotFound(e)).toBe(true);
    });

    it('returns false for det errors with not found not in message', () => {
      const e = new DetError(
        'could not do',
        { publicMessage: 'could not do because bad', silent: true },
      );
      expect(service.isNotFound(e)).toBe(false);

    });
    it('returns true for errors with not found in message', () => {
      const e = new Error('could not do because not found');
      expect(service.isNotFound(e)).toBe(true);
    });
    it('returns false for errors with not found not in message', () => {
      const e = new Error('could not do because bad');
      expect(service.isNotFound(e)).toBe(false);
    });
  });

  describe('isAborted', () => {
    it('returns false for non errors', () => {
      const response = new Response('ok', { status: 200 });
      expect(service.isAborted(response)).toBe(false);
    });

    it('returns false for non-abort errors', () => {
      const error = new Error('could not do');
      expect(service.isAborted(error)).toBe(false);
    });

    it('returns true for abort errors', () => {
      const error = new Error('the operation was aborted');
      error.name = 'AbortError';
      expect(service.isAborted(error)).toBe(true);
    });
  });

  describe('processApiError', () => {
    it('is good', () => {

      expect(1).toBe(1);
    });
  });

  describe('generateDetApi', () => {
    it('iis good', () => {

      expect(1).toBe(1);
    });
  });

  describe('validateDetApiEnum', () => {
    it('returns valid enum values', () => {
      expect(service.validateDetApiEnum(V1OrderBy, V1OrderBy.ASC)).toBe(V1OrderBy.ASC);
    });

    it('returns valid string values', () => {
      expect(service.validateDetApiEnum(V1OrderBy, 'ORDER_BY_ASC')).toBe(V1OrderBy.ASC);
    });

    it('returns default for invalid values', () => {
      expect(service.validateDetApiEnum(V1OrderBy, 'asdfasdf'))
        .toBe(V1OrderBy.UNSPECIFIED);
    });

    it('returns undefined when no default value exists', () => {
      expect(service.validateDetApiEnum(ProtobufNullValue, 'asdfasdf'))
        .toBeUndefined();
    });
  });

  describe('validateDetApiEnumList', () => {
    it('should preserve valid input list', () => {
      const input = [ V1OrderBy.ASC, V1OrderBy.DESC ];
      const expectedOutput = [ V1OrderBy.ASC, V1OrderBy.DESC ];
      expect(service.validateDetApiEnumList(V1OrderBy, input)).toStrictEqual(expectedOutput);
    });

    it('should return undefined when all inputs are unspecified or invalid', () => {
      const input = [ V1OrderBy.UNSPECIFIED, V1OrderBy.UNSPECIFIED, 'bucket' ];

      expect(service.validateDetApiEnumList(V1OrderBy, input)).toBeUndefined();
    });

    it('should filter bad entries', () => {
      const input = [ V1OrderBy.ASC, V1OrderBy.DESC, 'bucket' ];
      const expectedOutput = [ V1OrderBy.ASC, V1OrderBy.DESC ];
      expect(service.validateDetApiEnumList(V1OrderBy, input)).toStrictEqual(expectedOutput);
    });
  });

  describe('noOp', () => {
    it('does nothing', () => {
      expect(service.noOp()).toBeUndefined();
    });
  });

  // export const identity = <T>(a: T): T => a;
  describe('identity', () => {
    it('passes the same thing back', () => {
      Object.values(window).forEach((thing) => {
        expect(service.identity(thing)).toBe(thing);
      });

    });
  });
});
