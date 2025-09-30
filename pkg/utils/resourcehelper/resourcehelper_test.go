package resourcehelper

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestIsScalarResourceName(t *testing.T) {
	type args struct {
		name corev1.ResourceName
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Is ScalarResourceName",
			args: args{
				name: "attachable-volumes-demo",
			},
			want: true,
		},
		{
			name: "Is not IsScalarResourceName",
			args: args{
				name: "kubernetes-demo",
			},
			want: false,
		},
		{
			name: "Is ScalarResourceName",
			args: args{
				name: "kubernetes.io/",
			},
			want: true,
		},
		{
			name: "Is not ScalarResourceName",
			args: args{
				name: "kubernetes",
			},
			want: false,
		},
		{
			name: "Is not ScalarResourceName",
			args: args{
				name: "requests/",
			},
			want: false,
		},
		{
			name: "IsScalarResourceName",
			args: args{
				name: "requests/kpanda.io",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsScalarResourceName(tt.args.name); got != tt.want {
				t.Errorf("IsScalarResourceName(%s) = %v, want %v", tt.args.name, got, tt.want)
			}
		})
	}
}
