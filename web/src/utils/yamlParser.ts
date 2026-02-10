import yaml from 'js-yaml';

// YAML parser using standard js-yaml library
// Returns untyped structure since YAML content is dynamic and not validated
export const parseSimpleYaml = (yamlText: string): Record<string, unknown> => {
  try {
    const parsed = yaml.load(yamlText);
    return (parsed || {}) as Record<string, unknown>;
  } catch (error) {
    console.error('YAML parsing error:', error);
    return {};
  }
};