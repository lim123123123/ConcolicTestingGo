package functionflow

import "go.uber.org/zap"

type functionFlowAnalyzerConfig struct {
	DumpSSAToFile string // Path to the file
	Logger        *zap.SugaredLogger
}

type FunctionFlowAnalyzerOption func(*functionFlowAnalyzerConfig)

func WithDumpSSAToFile(filePath string) FunctionFlowAnalyzerOption {
	return func(ffac *functionFlowAnalyzerConfig) {
		ffac.DumpSSAToFile = filePath
	}
}

func WithLogger(logger *zap.SugaredLogger) FunctionFlowAnalyzerOption {
	return func(ffac *functionFlowAnalyzerConfig) {
		ffac.Logger = logger
	}
}
