package cli

import (
	"fmt"
	"strconv"

	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/spf13/cobra"
)

type PipelineCLI struct {
	usecase usecase.PipelineUseCase
}

func NewPipelineCLI(uc usecase.PipelineUseCase) *PipelineCLI {
	return &PipelineCLI{usecase: uc}
}

func (c *PipelineCLI) CreatePipelineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new pipeline",
		Run: func(cmd *cobra.Command, args []string) {
			targetsystemid, _ := cmd.Flags().GetInt("targetsystemid")
			route, _ := cmd.Flags().GetString("route")
			status, _ := cmd.Flags().GetString("status")
			currentstage, _ := cmd.Flags().GetString("currentstage")

			input := usecase.CreatePipelineInput{
				TargetSystemID: targetsystemid,
				Route:          route,
				Status:         status,
				CurrentStage:   currentstage,
			}

			output, err := c.usecase.CreatePipeline(input)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			fmt.Printf("Pipeline created: %+v\n", output)
		},
	}

	cmd.Flags().Int("targetsystemid", 0, "TargetSystemID of the pipeline")
	cmd.Flags().String("route", "", "Route of the pipeline")
	cmd.Flags().String("status", "", "Status of the pipeline")
	cmd.Flags().String("currentstage", "", "CurrentStage of the pipeline")
	return cmd
}

func (c *PipelineCLI) GetPipelineCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get [id]",
		Short: "Get pipeline by ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid ID: %v\n", err)
				return
			}

			pipeline, err := c.usecase.GetPipeline(id)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			fmt.Printf("Pipeline: %+v\n", pipeline)
		},
	}
}
