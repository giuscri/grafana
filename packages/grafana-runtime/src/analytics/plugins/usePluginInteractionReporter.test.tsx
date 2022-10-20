import { renderHook } from '@testing-library/react-hooks';
import React from 'react';

import {
  DataSourceInstanceSettings,
  DataSourcePluginContextProvider,
  PluginContextProvider,
  PluginMeta,
  PluginSignatureStatus,
  PluginType,
} from '@grafana/data';

import { reportInteraction } from '../utils';

import { usePluginInteractionReporter } from './usePluginInteractionReporter';

jest.mock('../utils', () => ({ reportInteraction: jest.fn() }));
const reportInteractionMock = jest.mocked(reportInteraction);

describe('usePluginInteractionReporter', () => {
  beforeEach(() => jest.resetAllMocks());

  describe('within a panel plugin', () => {
    it('should report interaction with plugin context information', () => {
      const report = renderPluginReporterHook({});

      report('grafana_plugin_select_query_type');
      expect(reportInteractionMock.mock.calls.length).toBe(1);
    });
  });

  describe('within a data source plugin', () => {
    it('should report interaction with plugin context information', () => {
      const report = renderDataSourcePluginReporterHook();
      report('grafana_plugin_select_query_type');

      expect(reportInteractionMock.mock.calls.length).toBe(1);
    });
  });

  describe('ensure interaction name follows convention', () => {
    it('should throw name does not start with "grafana_plugin_"', () => {
      const report = renderDataSourcePluginReporterHook();
      expect(() => report('select_query_type')).toThrow();
    });

    it('should throw if name is exactly "grafana_plugin_"', () => {
      const report = renderPluginReporterHook();
      expect(() => report('grafana_plugin_')).toThrow();
    });
  });
});

function renderPluginReporterHook(meta?: Partial<PluginMeta>): typeof reportInteraction {
  const wrapper = ({ children }: React.PropsWithChildren<{}>) => (
    <PluginContextProvider meta={createPluginMeta(meta)}>{children}</PluginContextProvider>
  );
  const { result } = renderHook(() => usePluginInteractionReporter(), { wrapper });
  return result.current;
}

function renderDataSourcePluginReporterHook(settings?: Partial<DataSourceInstanceSettings>): typeof reportInteraction {
  const wrapper = ({ children }: React.PropsWithChildren<{}>) => (
    <DataSourcePluginContextProvider instanceSettings={createDataSourceInstanceSettings(settings)}>
      {children}
    </DataSourcePluginContextProvider>
  );
  const { result } = renderHook(() => usePluginInteractionReporter(), { wrapper });
  return result.current;
}

function createPluginMeta(partial: Partial<PluginMeta> = {}): PluginMeta {
  const { info, ...rest } = partial;

  return {
    id: 'gauge',
    name: 'Gauge',
    type: PluginType.panel,
    info: {
      author: { name: 'Grafana Labs' },
      description: 'Standard gauge visualization',
      links: [],
      logos: {
        large: 'public/app/plugins/panel/gauge/img/icon_gauge.svg',
        small: 'public/app/plugins/panel/gauge/img/icon_gauge.svg',
      },
      screenshots: [],
      updated: '',
      version: '',
      ...info,
    },
    module: 'app/plugins/panel/gauge/module',
    baseUrl: '',
    signature: PluginSignatureStatus.internal,
    ...rest,
  };
}

function createDataSourceInstanceSettings(
  settings: Partial<DataSourceInstanceSettings> = {}
): DataSourceInstanceSettings {
  const { meta, ...rest } = settings;

  return {
    id: 1,
    uid: '',
    name: '',
    meta: createPluginMeta(meta),
    type: PluginType.datasource,
    readOnly: false,
    jsonData: {},
    access: 'proxy',
    ...rest,
  };
}
