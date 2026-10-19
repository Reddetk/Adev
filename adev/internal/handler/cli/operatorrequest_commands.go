package cli

import (
	"fmt"
	"strconv"

	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/spf13/cobra"
)

type OperatorRequestCLI struct {
	usecase usecase.OperatorRequestUseCase
}

func NewOperatorRequestCLI(uc usecase.OperatorRequestUseCase) *OperatorRequestCLI {
	return &OperatorRequestCLI{usecase: uc}
}

func (c *OperatorRequestCLI) CreateOperatorRequestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new operatorrequest",
		Run: func(cmd *cobra.Command, args []string) {
			pipelineid, _ := cmd.Flags().GetInt("pipelineid")
			pointer, _ := cmd.Flags().GetString("pointer")
			question, _ := cmd.Flags().GetString("question")
			blocking, _ := cmd.Flags().GetBool("blocking")
			status, _ := cmd.Flags().GetString("status")

			input := usecase.CreateOperatorRequestInput{
				PipelineID: pipelineid,
				Pointer:    pointer,
				Question:   question,
				Blocking:   blocking,
				Status:     status,
			}

			output, err := c.usecase.CreateOperatorRequest(input)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			fmt.Printf("OperatorRequest created: %+v\n", output)
		},
	}

	cmd.Flags().Int("pipelineid", 0, "PipelineID of the operatorrequest")
	cmd.Flags().String("pointer", "", "Pointer of the operatorrequest")
	cmd.Flags().String("question", "", "Question of the operatorrequest")
	cmd.Flags().Bool("blocking", false, "Blocking of the operatorrequest")
	cmd.Flags().String("status", "", "Status of the operatorrequest")
	return cmd
}

func (c *OperatorRequestCLI) GetOperatorRequestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get [id]",
		Short: "Get operatorrequest by ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid ID: %v\n", err)
				return
			}

			operatorrequest, err := c.usecase.GetOperatorRequest(id)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			fmt.Printf("OperatorRequest: %+v\n", operatorrequest)
		},
	}
}
